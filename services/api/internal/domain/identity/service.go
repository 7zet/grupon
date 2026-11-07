package identity

import (
	"context"
	"errors"
	"log/slog"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	platformmail "github.com/grupon/grupon/services/api/internal/platform/mail"
)

// Lifetimes and branding used by the service.
type Config struct {
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	VerifyEmailTTL  time.Duration
	// WebBaseURL is the public web origin used to build links in emails, without a trailing slash.
	WebBaseURL string
	// OTPEcho returns the sign-in code in the response. Only development may enable it.
	OTPEcho bool
	// OTPCooldown is the minimum wait between two codes for one phone. Development sets it to zero so
	// automated sign-ins can repeat; other environments keep the full minute.
	OTPCooldown time.Duration
	// BrandName is substituted into email subjects.
	BrandName         string
	RealSMS           bool
	SMSDailyCapPerIP  int
	SMSDailyCapGlobal int
	// RateLimitFailClosed refuses requests when the limiter is missing or fails. Production sets it.
	RateLimitFailClosed bool
}

// Dependencies are the ports the service needs.
type Dependencies struct {
	Repository Repository
	Tokens     *TokenIssuer
	Mailer     platformmail.Sender
	Limiter    RateLimiter
	CodeSender CodeSender
	Logger     *slog.Logger
	Now        func() time.Time
}

// Service implements account and session rules. It has no HTTP or SQL knowledge.
type Service struct {
	repo       Repository
	tokens     *TokenIssuer
	mailer     platformmail.Sender
	limiter    RateLimiter
	codeSender CodeSender
	logger     *slog.Logger
	now        func() time.Time
	cfg        Config
}

// NewService wires the identity rules to their ports.
func NewService(deps Dependencies, cfg Config) *Service {
	logger := deps.Logger
	if logger == nil {
		logger = slog.Default()
	}
	now := deps.Now
	if now == nil {
		now = time.Now
	}
	return &Service{
		repo:       deps.Repository,
		tokens:     deps.Tokens,
		mailer:     deps.Mailer,
		limiter:    deps.Limiter,
		codeSender: deps.CodeSender,
		logger:     logger,
		now:        now,
		cfg:        cfg,
	}
}

// AccessTokenTTL returns the lifetime of access tokens, reported to clients as expires_in.
func (s *Service) AccessTokenTTL() time.Duration { return s.cfg.AccessTokenTTL }

// RegisterInput is the validated-at-the-edge registration payload.
type RegisterInput struct {
	Email    string
	Password string
	Name     string
	Locale   string
	Currency string
}

// Refresh rotates a refresh token. Presenting a token that was already rotated or revoked revokes
// the whole family, because it indicates that the token was copied.
func (s *Service) Refresh(ctx context.Context, meta RequestMeta, raw string) (Session, error) {
	if raw == "" {
		return Session{}, ErrRefreshTokenInvalid
	}
	current, err := s.repo.GetRefreshTokenByHash(ctx, HashToken(raw))
	if errors.Is(err, ErrNotFound) {
		return Session{}, ErrRefreshTokenInvalid
	}
	if err != nil {
		return Session{}, err
	}
	if current.RevokedAt != nil {
		if err := s.repo.RevokeRefreshFamily(ctx, current.FamilyID); err != nil {
			return Session{}, err
		}
		s.logger.WarnContext(ctx, "refresh token reuse detected; family revoked")
		return Session{}, ErrRefreshTokenReused
	}
	now := s.now()
	if !now.Before(current.ExpiresAt) {
		return Session{}, ErrRefreshTokenInvalid
	}
	user, err := s.repo.GetUserByID(ctx, current.UserID)
	if errors.Is(err, ErrNotFound) {
		return Session{}, ErrRefreshTokenInvalid
	}
	if err != nil {
		return Session{}, err
	}
	if user.Status == StatusBlocked {
		return Session{}, ErrAccountBlocked
	}

	next, err := s.newRefreshToken(user.ID, current.FamilyID, meta)
	if err != nil {
		return Session{}, err
	}
	// Rotation never extends the original session boundary. Privileged sessions are therefore
	// capped at eight hours even when the global refresh-token TTL is longer.
	next.row.ExpiresAt = current.ExpiresAt
	if err := s.repo.RotateRefreshToken(ctx, current.ID, next.row); err != nil {
		if errors.Is(err, ErrRefreshTokenReused) {
			if revokeErr := s.repo.RevokeRefreshFamily(ctx, current.FamilyID); revokeErr != nil {
				return Session{}, revokeErr
			}
		}
		return Session{}, err
	}
	return s.buildSession(ctx, user, next.raw)
}

// Logout revokes the presented refresh token. Unknown or empty tokens are not an error.
func (s *Service) Logout(ctx context.Context, raw string) error {
	if raw == "" {
		return nil
	}
	return s.repo.RevokeRefreshTokenByHash(ctx, HashToken(raw))
}

// Authenticate resolves an access token to a live account with fresh roles.
func (s *Service) Authenticate(ctx context.Context, accessToken string) (Account, error) {
	if accessToken == "" {
		return Account{}, ErrUnauthenticated
	}
	userID, err := s.tokens.Parse(accessToken)
	if err != nil {
		return Account{}, ErrUnauthenticated
	}
	user, err := s.repo.GetUserByID(ctx, userID)
	if errors.Is(err, ErrNotFound) {
		return Account{}, ErrUnauthenticated
	}
	if err != nil {
		return Account{}, err
	}
	if user.Status == StatusBlocked {
		return Account{}, ErrAccountBlocked
	}
	return s.account(ctx, user)
}

// Profile returns the account with its roles.
func (s *Service) Profile(ctx context.Context, userID uuid.UUID) (Account, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return Account{}, err
	}
	return s.account(ctx, user)
}

// UpdateProfile applies the provided profile fields. An empty phone clears it.
func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, update ProfileUpdate) (Account, error) {
	if update.Name != nil {
		name := strings.TrimSpace(*update.Name)
		if err := validateName(name); err != nil {
			return Account{}, err
		}
		update.Name = &name
	}
	if update.Phone != nil {
		phone := strings.TrimSpace(*update.Phone)
		update.Phone = &phone
	}
	if update.Locale != nil && !LocaleValid(*update.Locale) {
		return Account{}, &ValidationError{Field: "locale", Code: "unsupported"}
	}
	if update.Currency != nil && !CurrencyValid(*update.Currency) {
		return Account{}, &ValidationError{Field: "currency", Code: "unsupported"}
	}
	user, err := s.repo.UpdateProfile(ctx, userID, update)
	if err != nil {
		return Account{}, err
	}
	return s.account(ctx, user)
}

// DeleteAccount soft-deletes the user and revokes all sessions.
func (s *Service) DeleteAccount(ctx context.Context, userID uuid.UUID) error {
	if err := s.repo.SoftDeleteUser(ctx, userID); err != nil {
		return err
	}
	return s.repo.RevokeUserSessions(ctx, userID, nil)
}

type newRefresh struct {
	raw string
	row RefreshToken
}

func (s *Service) newRefreshToken(userID, familyID uuid.UUID, meta RequestMeta) (newRefresh, error) {
	raw, err := NewOpaqueToken()
	if err != nil {
		return newRefresh{}, err
	}
	return newRefresh{
		raw: raw,
		row: RefreshToken{
			UserID:    userID,
			TokenHash: HashToken(raw),
			FamilyID:  familyID,
			ExpiresAt: s.now().Add(s.cfg.RefreshTokenTTL),
			UserAgent: meta.UserAgent,
			IP:        meta.IP,
		},
	}, nil
}

// openSession starts a new refresh-token family and returns a session.
func (s *Service) openSession(ctx context.Context, meta RequestMeta, user User, familyID uuid.UUID) (Session, error) {
	next, err := s.newRefreshToken(user.ID, familyID, meta)
	if err != nil {
		return Session{}, err
	}
	roles, err := s.repo.ListRoles(ctx, user.ID)
	if err != nil {
		return Session{}, err
	}
	if _, required := privileged(roles); required {
		limit := s.now().Add(8 * time.Hour)
		if next.row.ExpiresAt.After(limit) {
			next.row.ExpiresAt = limit
		}
	}
	if err := s.repo.CreateRefreshToken(ctx, next.row); err != nil {
		return Session{}, err
	}
	return s.buildSession(ctx, user, next.raw)
}

func (s *Service) buildSession(ctx context.Context, user User, refreshRaw string) (Session, error) {
	account, err := s.account(ctx, user)
	if err != nil {
		return Session{}, err
	}
	access, err := s.tokens.Issue(user.ID, account.Roles)
	if err != nil {
		return Session{}, err
	}
	return Session{Account: account, AccessToken: access, RefreshToken: refreshRaw}, nil
}

func (s *Service) account(ctx context.Context, user User) (Account, error) {
	roles, err := s.repo.ListRoles(ctx, user.ID)
	if err != nil {
		return Account{}, err
	}
	return Account{User: user, Roles: roles}, nil
}

const (
	kindVerifyEmail = "verify_email"
)

// allow spends one attempt on every key. A limiter failure is logged and the request is allowed
// outside production; with RateLimitFailClosed (production) it is refused.
func (s *Service) allow(ctx context.Context, keys ...string) error {
	if s.limiter == nil {
		if s.cfg.RateLimitFailClosed {
			return ErrRateLimited
		}
		return nil
	}
	for _, key := range keys {
		if key == "" {
			continue
		}
		ok, err := s.limiter.Allow(ctx, key)
		if err != nil {
			if s.cfg.RateLimitFailClosed {
				s.logger.ErrorContext(ctx, "rate limiter unavailable; refusing request", slogError(err))
				return ErrRateLimited
			}
			s.logger.WarnContext(ctx, "rate limiter unavailable; allowing request", slogError(err))
			continue
		}
		if !ok {
			return ErrRateLimited
		}
	}
	return nil
}

func ipKey(scope, ip string) string {
	if ip == "" {
		return ""
	}
	return scope + ":ip:" + ip
}

// hashedKey keeps personal data (emails) out of Redis keys.
func hashedKey(scope, value string) string {
	return scope + ":" + HashToken(value)
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validateEmail(email string) error {
	if len(email) > 320 {
		return &ValidationError{Field: "email", Code: "invalid"}
	}
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email {
		return &ValidationError{Field: "email", Code: "invalid"}
	}
	return nil
}

func validateName(name string) error {
	length := utf8.RuneCountInString(name)
	if length < 1 || length > 200 {
		return &ValidationError{Field: "name", Code: "invalid_length"}
	}
	return nil
}

func slogError(err error) slog.Attr {
	return slog.String("error", err.Error())
}
