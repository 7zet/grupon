package identity

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Phone sign-in rules. Codes are six digits, valid for five minutes, and a phone may request a new
// code at most once a minute. Five wrong guesses use up a code.
const (
	otpLength       = 6
	otpTTL          = 5 * time.Minute
	otpCooldown     = time.Minute
	otpMaxAttempts  = 5
	defaultUserName = "Foydalanuvchi"
	placeholderHost = "phone.grupon.local"
)

// Errors for phone sign-in. The HTTP layer maps them to problem codes.
var (
	ErrInvalidPhone    = errors.New("identity: phone is not a valid Uzbekistan number")
	ErrOTPCooldown     = errors.New("identity: a code was sent recently")
	ErrOTPInvalid      = errors.New("identity: code is wrong or expired")
	ErrOTPTooManyTries = errors.New("identity: too many wrong codes")
	ErrOTPUnavailable  = errors.New("identity: sign-in codes are not available right now")
)

var uzPhone = regexp.MustCompile(`^\+998[0-9]{9}$`)

// CodeSender delivers a sign-in code to a phone. Production uses an SMS provider; development logs it.
type CodeSender interface {
	SendCode(ctx context.Context, phone, code string) error
}

// OTPRequest is the result of asking for a code. DevCode is set only in development, so the
// test flow can finish without an SMS provider. It is never set in production.
type OTPRequest struct {
	CooldownSeconds int
	DevCode         string
}

// NormalizePhone returns the canonical +998XXXXXXXXX form, or ErrInvalidPhone.
func NormalizePhone(raw string) (string, error) {
	cleaned := strings.NewReplacer(" ", "", "-", "", "(", "", ")", "").Replace(strings.TrimSpace(raw))
	if !uzPhone.MatchString(cleaned) {
		return "", ErrInvalidPhone
	}
	return cleaned, nil
}

// RequestCode creates a new code for the phone and hands it to the sender.
func (s *Service) RequestCode(ctx context.Context, meta RequestMeta, rawPhone string) (OTPRequest, error) {
	phone, err := NormalizePhone(rawPhone)
	if err != nil {
		return OTPRequest{}, err
	}
	if err := s.allow(ctx, ipKey("otp", meta.IP), hashedKey("otp:phone", phone)); err != nil {
		return OTPRequest{}, err
	}
	if s.codeSender == nil {
		return OTPRequest{}, ErrOTPUnavailable
	}
	latest, err := s.repo.LatestIssuedPhoneOTP(ctx, phone)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return OTPRequest{}, err
	}
	if err == nil && s.cfg.OTPCooldown > 0 && s.now().Sub(latest.CreatedAt) < s.cfg.OTPCooldown {
		return OTPRequest{}, ErrOTPCooldown
	}

	code, err := newOTPCode()
	if err != nil {
		return OTPRequest{}, err
	}
	if err := s.repo.InsertPhoneOTP(ctx, phone, HashToken(phone+":"+code), s.now().Add(otpTTL)); err != nil {
		return OTPRequest{}, err
	}
	if err := s.codeSender.SendCode(ctx, phone, code); err != nil {
		return OTPRequest{}, fmt.Errorf("send code: %w", err)
	}
	result := OTPRequest{CooldownSeconds: int(s.cfg.OTPCooldown / time.Second)}
	if s.cfg.OTPEcho {
		result.DevCode = code
	}
	return result, nil
}

// VerifyCode checks the newest unused code for the phone. A correct code signs the account in and
// creates it on first use.
func (s *Service) VerifyCode(ctx context.Context, meta RequestMeta, rawPhone, code string) (Session, error) {
	user, err := s.verifyPhoneCode(ctx, meta, rawPhone, code)
	if err != nil {
		return Session{}, err
	}
	roles, err := s.repo.ListRoles(ctx, user.ID)
	if err != nil {
		return Session{}, err
	}
	if _, required := privileged(roles); required {
		return Session{}, ErrTwoFactorRequired
	}
	return s.openSession(ctx, meta, user, uuid.Must(uuid.NewV7()))
}

func (s *Service) verifyPhoneCode(ctx context.Context, meta RequestMeta, rawPhone, code string) (User, error) {
	phone, err := NormalizePhone(rawPhone)
	if err != nil {
		return User{}, err
	}
	if err := s.allow(ctx, ipKey("otp:verify", meta.IP), hashedKey("otp:verify:phone", phone)); err != nil {
		return User{}, err
	}
	latest, err := s.repo.LatestPhoneOTP(ctx, phone)
	if errors.Is(err, ErrNotFound) {
		return User{}, ErrOTPInvalid
	}
	if err != nil {
		return User{}, err
	}
	if !s.now().Before(latest.ExpiresAt) {
		return User{}, ErrOTPInvalid
	}
	if latest.Attempts >= otpMaxAttempts {
		return User{}, ErrOTPTooManyTries
	}
	if subtle.ConstantTimeCompare([]byte(HashToken(phone+":"+strings.TrimSpace(code))), []byte(latest.CodeHash)) != 1 {
		if err := s.repo.IncrementPhoneOTPAttempts(ctx, latest.ID); err != nil {
			return User{}, err
		}
		return User{}, ErrOTPInvalid
	}
	consumed, err := s.repo.ConsumePhoneOTP(ctx, latest.ID)
	if err != nil {
		return User{}, err
	}
	if !consumed {
		return User{}, ErrOTPInvalid
	}

	user, err := s.repo.GetUserByPhone(ctx, phone)
	if errors.Is(err, ErrNotFound) {
		user, err = s.repo.CreatePhoneUser(ctx, phone, placeholderEmail(phone))
	}
	if err != nil {
		return User{}, err
	}
	if user.Status == StatusBlocked {
		return User{}, ErrAccountBlocked
	}
	if err := s.repo.TouchLastLogin(ctx, user.ID); err != nil {
		return User{}, err
	}
	return user, nil
}

func newOTPCode() (string, error) {
	max := big.NewInt(1)
	for i := 0; i < otpLength; i++ {
		max.Mul(max, big.NewInt(10))
	}
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", fmt.Errorf("random code: %w", err)
	}
	return fmt.Sprintf("%0*d", otpLength, n), nil
}

// placeholderEmail gives phone-only accounts a unique address until email sign-in exists (R1b).
func placeholderEmail(phone string) string {
	return strings.TrimPrefix(phone, "+") + "@" + placeholderHost
}
