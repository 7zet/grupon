package identity

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1" // TOTP requires HMAC-SHA1 for broad authenticator compatibility.
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

const twoFactorChallengeTTL = 10 * time.Minute

var (
	ErrTwoFactorRequired = errors.New("identity: two factor authentication required")
	ErrTwoFactorInvalid  = errors.New("identity: invalid two factor code")
	ErrTwoFactorSetup    = errors.New("identity: two factor enrollment required")
)

func privileged(roles []Role) (Role, bool) {
	for _, role := range roles {
		switch role {
		case RoleAdmin, RoleModerator, RoleSupport, RoleFinance, RoleSuperadmin:
			return role, true
		}
	}
	return "", false
}

// VerifyPhone starts mandatory 2FA for platform staff and completes ordinary sign-ins directly.
func (s *Service) VerifyPhone(ctx context.Context, meta RequestMeta, phone, code string) (PhoneVerification, error) {
	user, err := s.verifyPhoneCode(ctx, meta, phone, code)
	if err != nil {
		return PhoneVerification{}, err
	}
	roles, err := s.repo.ListRoles(ctx, user.ID)
	if err != nil {
		return PhoneVerification{}, err
	}
	if _, required := privileged(roles); !required {
		session, err := s.openSession(ctx, meta, user, uuid.Must(uuid.NewV7()))
		if err != nil {
			return PhoneVerification{}, err
		}
		return PhoneVerification{Session: &session}, nil
	}
	state, err := s.repo.GetTwoFactor(ctx, user.ID)
	enrollment := errors.Is(err, ErrNotFound) || !state.Enabled
	if err != nil && !errors.Is(err, ErrNotFound) {
		return PhoneVerification{}, err
	}
	raw, err := NewOpaqueToken()
	if err != nil {
		return PhoneVerification{}, err
	}
	if err := s.repo.CreateTwoFactorChallenge(ctx, user.ID, HashToken(raw), s.now().Add(twoFactorChallengeTTL)); err != nil {
		return PhoneVerification{}, err
	}
	return PhoneVerification{Challenge: &TwoFactorChallenge{Token: raw, Enrollment: enrollment}}, nil
}

// BeginTwoFactor creates fresh authenticator setup material for a valid login challenge.
func (s *Service) BeginTwoFactor(ctx context.Context, challenge string) (TwoFactorEnrollment, error) {
	userID, err := s.challengeUser(ctx, challenge)
	if err != nil {
		return TwoFactorEnrollment{}, err
	}
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return TwoFactorEnrollment{}, err
	}
	secretBytes := make([]byte, 20)
	if _, err := rand.Read(secretBytes); err != nil {
		return TwoFactorEnrollment{}, fmt.Errorf("generate TOTP secret: %w", err)
	}
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secretBytes)
	if err := s.repo.BeginTwoFactorEnrollment(ctx, userID, secret); err != nil {
		return TwoFactorEnrollment{}, err
	}
	label := url.QueryEscape(s.cfg.BrandName + ":" + user.Email)
	issuer := url.QueryEscape(s.cfg.BrandName)
	return TwoFactorEnrollment{Secret: secret, OTPAuthURI: "otpauth://totp/" + label + "?secret=" + secret + "&issuer=" + issuer + "&digits=6&period=30"}, nil
}

// EnableTwoFactor verifies setup, stores ten one-time codes, audits, and opens the staff session.
func (s *Service) EnableTwoFactor(ctx context.Context, meta RequestMeta, challenge, code string) (Session, []string, error) {
	userID, err := s.challengeUser(ctx, challenge)
	if err != nil {
		return Session{}, nil, err
	}
	state, err := s.repo.GetTwoFactor(ctx, userID)
	if err != nil || state.Enabled || !validTOTP(state.Secret, code, s.now()) {
		return Session{}, nil, ErrTwoFactorInvalid
	}
	account, err := s.Profile(ctx, userID)
	if err != nil {
		return Session{}, nil, err
	}
	role, ok := privileged(account.Roles)
	if !ok {
		return Session{}, nil, ErrTwoFactorInvalid
	}
	codes, hashes, err := recoveryCodes()
	if err != nil {
		return Session{}, nil, err
	}
	if err := s.repo.EnableTwoFactor(ctx, userID, hashes, role); err != nil {
		return Session{}, nil, err
	}
	if ok, err := s.repo.ConsumeTwoFactorChallenge(ctx, HashToken(challenge)); err != nil || !ok {
		return Session{}, nil, ErrTwoFactorInvalid
	}
	session, err := s.openSession(ctx, meta, account.User, uuid.Must(uuid.NewV7()))
	return session, codes, err
}

// CompleteTwoFactor accepts an authenticator or one-time recovery code.
func (s *Service) CompleteTwoFactor(ctx context.Context, meta RequestMeta, challenge, code string) (Session, error) {
	userID, err := s.challengeUser(ctx, challenge)
	if err != nil {
		return Session{}, err
	}
	state, err := s.repo.GetTwoFactor(ctx, userID)
	if err != nil || !state.Enabled {
		return Session{}, ErrTwoFactorSetup
	}
	account, err := s.Profile(ctx, userID)
	if err != nil {
		return Session{}, err
	}
	role, _ := privileged(account.Roles)
	valid := validTOTP(state.Secret, code, s.now())
	if !valid {
		valid, err = s.repo.UseRecoveryCode(ctx, userID, HashToken(strings.ToUpper(strings.TrimSpace(code))), role)
		if err != nil {
			return Session{}, err
		}
	}
	if !valid {
		return Session{}, ErrTwoFactorInvalid
	}
	if ok, err := s.repo.ConsumeTwoFactorChallenge(ctx, HashToken(challenge)); err != nil || !ok {
		return Session{}, ErrTwoFactorInvalid
	}
	return s.openSession(ctx, meta, account.User, uuid.Must(uuid.NewV7()))
}

func (s *Service) DisableTwoFactor(ctx context.Context, account Account, code string) error {
	role, ok := privileged(account.Roles)
	if !ok {
		return ErrTwoFactorInvalid
	}
	state, err := s.repo.GetTwoFactor(ctx, account.ID)
	if err != nil {
		return err
	}
	if !validTOTP(state.Secret, code, s.now()) {
		return ErrTwoFactorInvalid
	}
	return s.repo.DisableTwoFactor(ctx, account.ID, role)
}

func (s *Service) challengeUser(ctx context.Context, raw string) (uuid.UUID, error) {
	if raw == "" {
		return uuid.Nil, ErrTwoFactorInvalid
	}
	return s.repo.GetTwoFactorChallenge(ctx, HashToken(raw))
}

func validTOTP(secret, code string, now time.Time) bool {
	code = strings.TrimSpace(code)
	for delta := int64(-1); delta <= 1; delta++ {
		if hmac.Equal([]byte(totpAt(secret, now.Unix()/30+delta)), []byte(code)) {
			return true
		}
	}
	return false
}

func totpAt(secret string, counter int64) string {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return ""
	}
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], uint64(counter))
	h := hmac.New(sha1.New, key)
	_, _ = h.Write(buf[:])
	sum := h.Sum(nil)
	offset := sum[len(sum)-1] & 15
	value := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff
	return fmt.Sprintf("%06d", value%1000000)
}

func recoveryCodes() ([]string, []string, error) {
	codes := make([]string, 10)
	hashes := make([]string, 10)
	for i := range codes {
		b := make([]byte, 5)
		if _, err := rand.Read(b); err != nil {
			return nil, nil, err
		}
		codes[i] = strings.ToUpper(fmt.Sprintf("%X-%X", b[:2], b[2:]))
		hashes[i] = HashToken(codes[i])
	}
	return codes, hashes, nil
}
