package identity

import (
	"context"
	"errors"
	"fmt"
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
	AccessTokenTTL   time.Duration
	RefreshTokenTTL  time.Duration
	VerifyEmailTTL   time.Duration
	ResetPasswordTTL time.Duration
	// WebBaseURL is the public web origin used to build links in emails, without a trailing slash.
	WebBaseURL string
	// BrandName is substituted into email subjects.
	BrandName string
}

// Dependencies are the ports the service needs. Google may be nil when sign-in is not configured.
type Dependencies struct {
	Repository Repository
	Tokens     *TokenIssuer
	Mailer     platformmail.Sender
	Limiter    RateLimiter
	Google     GoogleExchanger
	Logger     *slog.Logger
	Now        func() time.Time
}

// Service implements account and session rules. It has no HTTP or SQL knowledge.
type Service struct {
	repo    Repository
	tokens  *TokenIssuer
	mailer  platformmail.Sender
	limiter RateLimiter
	google  GoogleExchanger
	logger  *slog.Logger
	now     func() time.Time
	cfg     Config
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
		repo:    deps.Repository,
		tokens:  deps.Tokens,
		mailer:  deps.Mailer,
		limiter: deps.Limiter,
		google:  deps.Google,
		logger:  logger,
		now:     now,
		cfg:     cfg,
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

// Register creates a user with the user role, sends a verification email and opens a session.
func (s *Service) Register(ctx context.Context, meta RequestMeta, in RegisterInput) (Session, error) {
	email := normalizeEmail(in.Email)
	if err := s.allow(ctx, ipKey("register", meta.IP), hashedKey("register:email", email)); err != nil {
		return Session{}, err
	}
	if err := validateEmail(email); err != nil {
		return Session{}, err
	}
	name := strings.TrimSpace(in.Name)
	if err := validateName(name); err != nil {
		return Session{}, err
	}
	if !LocaleValid(in.Locale) {
		return Session{}, &ValidationError{Field: "locale", Code: "unsupported"}
	}
	if !CurrencyValid(in.Currency) {
		return Session{}, &ValidationError{Field: "currency", Code: "unsupported"}
	}
	if err := ValidatePassword(in.Password); err != nil {
		return Session{}, err
	}
	hash, err := HashPassword(in.Password)
	if err != nil {
		return Session{}, err
	}

	user, err := s.repo.CreateUser(ctx, NewUser{
		Email:        email,
		PasswordHash: &hash,
		Name:         name,
		Locale:       in.Locale,
		Currency:     in.Currency,
	})
	if err != nil {
		return Session{}, err
	}
	s.sendVerification(ctx, user)
	return s.openSession(ctx, meta, user, uuid.Must(uuid.NewV7()))
}

// Login checks email and password. Unknown emails, wrong passwords and password-less accounts all
// return ErrInvalidCredentials after the same amount of hashing work.
func (s *Service) Login(ctx context.Context, meta RequestMeta, email, password string) (Session, error) {
	normalized := normalizeEmail(email)
	if err := s.allow(ctx, ipKey("login", meta.IP), hashedKey("login:email", normalized)); err != nil {
		return Session{}, err
	}

	user, err := s.repo.GetUserByEmail(ctx, normalized)
	if errors.Is(err, ErrNotFound) {
		equalizeTiming(password)
		return Session{}, ErrInvalidCredentials
	}
	if err != nil {
		return Session{}, err
	}
	if user.PasswordHash == nil {
		equalizeTiming(password)
		return Session{}, ErrInvalidCredentials
	}
	ok, err := VerifyPassword(password, *user.PasswordHash)
	if err != nil {
		return Session{}, fmt.Errorf("verify password: %w", err)
	}
	if !ok {
		return Session{}, ErrInvalidCredentials
	}
	if user.Status == StatusBlocked {
		return Session{}, ErrAccountBlocked
	}
	if err := s.repo.TouchLastLogin(ctx, user.ID); err != nil {
		return Session{}, err
	}
	return s.openSession(ctx, meta, user, uuid.Must(uuid.NewV7()))
}

// GoogleSignInInput is the PKCE authorization-code exchange request.
type GoogleSignInInput struct {
	Code         string
	CodeVerifier string
	RedirectURI  string
}

// GoogleSignIn signs in with a Google account. A user is matched by google_sub first, then by a
// verified email; a matched email account is linked to the Google subject.
func (s *Service) GoogleSignIn(ctx context.Context, meta RequestMeta, in GoogleSignInInput) (Session, error) {
	if s.google == nil {
		return Session{}, ErrGoogleNotConfigured
	}
	if err := s.allow(ctx, ipKey("google", meta.IP)); err != nil {
		return Session{}, err
	}
	identity, err := s.google.Exchange(ctx, in.Code, in.CodeVerifier, in.RedirectURI)
	if err != nil {
		return Session{}, err
	}
	if identity.Subject == "" {
		return Session{}, ErrGoogleExchangeFailed
	}

	user, err := s.repo.GetUserByGoogleSub(ctx, identity.Subject)
	switch {
	case err == nil:
	case errors.Is(err, ErrNotFound):
		user, err = s.googleAccountForEmail(ctx, identity)
		if err != nil {
			return Session{}, err
		}
	default:
		return Session{}, err
	}
	if user.Status == StatusBlocked {
		return Session{}, ErrAccountBlocked
	}
	if err := s.repo.TouchLastLogin(ctx, user.ID); err != nil {
		return Session{}, err
	}
	return s.openSession(ctx, meta, user, uuid.Must(uuid.NewV7()))
}

func (s *Service) googleAccountForEmail(ctx context.Context, identity GoogleIdentity) (User, error) {
	if !identity.EmailVerified {
		return User{}, ErrGoogleEmailUnverified
	}
	email := normalizeEmail(identity.Email)
	if err := validateEmail(email); err != nil {
		return User{}, ErrGoogleExchangeFailed
	}

	existing, err := s.repo.GetUserByEmail(ctx, email)
	switch {
	case err == nil:
		if existing.GoogleSub != nil {
			return User{}, ErrGoogleAccountConflict
		}
		if err := s.repo.LinkGoogleSub(ctx, existing.ID, identity.Subject); err != nil {
			return User{}, err
		}
		if err := s.repo.MarkEmailVerified(ctx, existing.ID); err != nil {
			return User{}, err
		}
		existing.GoogleSub = &identity.Subject
		return existing, nil
	case errors.Is(err, ErrNotFound):
	default:
		return User{}, err
	}

	name := strings.TrimSpace(identity.Name)
	if name == "" || utf8.RuneCountInString(name) > 200 {
		name = strings.SplitN(email, "@", 2)[0]
	}
	sub := identity.Subject
	return s.repo.CreateUser(ctx, NewUser{
		Email:         email,
		GoogleSub:     &sub,
		Name:          name,
		Locale:        "uz",
		Currency:      "UZS",
		EmailVerified: true,
	})
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

// VerifyEmail consumes a verification token and marks the email as verified.
func (s *Service) VerifyEmail(ctx context.Context, raw string) error {
	if raw == "" {
		return ErrInvalidToken
	}
	userID, err := s.repo.ConsumeAuthToken(ctx, kindVerifyEmail, HashToken(raw))
	if err != nil {
		return err
	}
	return s.repo.MarkEmailVerified(ctx, userID)
}

// ForgotPassword sends a reset link when the email belongs to an active account. The result is the
// same for every email, so callers cannot tell which accounts exist.
func (s *Service) ForgotPassword(ctx context.Context, meta RequestMeta, email string) error {
	normalized := normalizeEmail(email)
	if err := s.allow(ctx, ipKey("forgot", meta.IP), hashedKey("forgot:email", normalized)); err != nil {
		return err
	}
	user, err := s.repo.GetUserByEmail(ctx, normalized)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if user.Status == StatusBlocked {
		return nil
	}
	if err := s.repo.InvalidateAuthTokens(ctx, user.ID, kindResetPassword); err != nil {
		return err
	}
	raw, err := NewOpaqueToken()
	if err != nil {
		return err
	}
	if err := s.repo.CreateAuthToken(ctx, user.ID, kindResetPassword, HashToken(raw), s.now().Add(s.cfg.ResetPasswordTTL)); err != nil {
		return err
	}
	link := fmt.Sprintf("%s/%s/reset-password?token=%s", s.cfg.WebBaseURL, user.Locale, raw)
	if err := s.mailer.Send(ctx, resetMail(s.cfg.BrandName, user.Locale, user.Email, link)); err != nil {
		s.logger.ErrorContext(ctx, "send password reset email failed", slogError(err))
	}
	return nil
}

// ResetPassword consumes a reset token, sets the new password, and revokes every session.
func (s *Service) ResetPassword(ctx context.Context, meta RequestMeta, raw, password string) error {
	if err := s.allow(ctx, ipKey("reset", meta.IP)); err != nil {
		return err
	}
	if err := ValidatePassword(password); err != nil {
		return err
	}
	if raw == "" {
		return ErrInvalidToken
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	userID, err := s.repo.ConsumeAuthToken(ctx, kindResetPassword, HashToken(raw))
	if err != nil {
		return err
	}
	if err := s.repo.UpdatePasswordHash(ctx, userID, hash); err != nil {
		return err
	}
	// The reset link was delivered to the mailbox, so the email is verified by it.
	if err := s.repo.MarkEmailVerified(ctx, userID); err != nil {
		return err
	}
	return s.repo.RevokeUserSessions(ctx, userID, nil)
}

// ChangePassword verifies the current password and sets a new one. Other sessions are revoked;
// the session that presented keepRefresh (may be empty) stays signed in.
func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, current, next, keepRefresh string) error {
	if err := s.allow(ctx, hashedKey("password:user", userID.String())); err != nil {
		return err
	}
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return ErrUnauthenticated
	}
	if user.PasswordHash == nil {
		return ErrCurrentPasswordWrong
	}
	ok, err := VerifyPassword(current, *user.PasswordHash)
	if err != nil {
		return fmt.Errorf("verify password: %w", err)
	}
	if !ok {
		return ErrCurrentPasswordWrong
	}
	if err := ValidatePassword(next); err != nil {
		return err
	}
	hash, err := HashPassword(next)
	if err != nil {
		return err
	}
	if err := s.repo.UpdatePasswordHash(ctx, userID, hash); err != nil {
		return err
	}

	var keep *uuid.UUID
	if keepRefresh != "" {
		token, err := s.repo.GetRefreshTokenByHash(ctx, HashToken(keepRefresh))
		if err == nil && token.UserID == userID {
			family := token.FamilyID
			keep = &family
		}
	}
	return s.repo.RevokeUserSessions(ctx, userID, keep)
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

func (s *Service) sendVerification(ctx context.Context, user User) {
	raw, err := NewOpaqueToken()
	if err == nil {
		err = s.repo.CreateAuthToken(ctx, user.ID, kindVerifyEmail, HashToken(raw), s.now().Add(s.cfg.VerifyEmailTTL))
	}
	if err != nil {
		s.logger.ErrorContext(ctx, "create verification token failed", slogError(err))
		return
	}
	link := fmt.Sprintf("%s/%s/verify-email?token=%s", s.cfg.WebBaseURL, user.Locale, raw)
	if err := s.mailer.Send(ctx, verificationMail(s.cfg.BrandName, user.Locale, user.Email, link)); err != nil {
		s.logger.ErrorContext(ctx, "send verification email failed", slogError(err))
	}
}

const (
	kindVerifyEmail   = "verify_email"
	kindResetPassword = "reset_password"
)

// allow spends one attempt on every key. A limiter failure is logged and the request is allowed,
// so a Redis outage does not lock every user out.
func (s *Service) allow(ctx context.Context, keys ...string) error {
	if s.limiter == nil {
		return nil
	}
	for _, key := range keys {
		if key == "" {
			continue
		}
		ok, err := s.limiter.Allow(ctx, key)
		if err != nil {
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
