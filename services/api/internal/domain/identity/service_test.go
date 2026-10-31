package identity_test

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/grupon/grupon/services/api/internal/domain/identity"
	"github.com/grupon/grupon/services/api/internal/domain/identity/identitytest"
)

const (
	testPassword = "correct-horse-battery"
	testEmail    = "buyer@example.test"
	testSecret   = "service-test-secret-with-at-least-32-bytes"
)

var tokenPattern = regexp.MustCompile(`token=([A-Za-z0-9_-]+)`)

type clock struct{ now time.Time }

func (c *clock) Now() time.Time          { return c.now }
func (c *clock) Advance(d time.Duration) { c.now = c.now.Add(d) }

type fixture struct {
	service *identity.Service
	repo    *identitytest.Repository
	mailer  *identitytest.Mailer
	clock   *clock
	google  *fakeGoogle
}

type fakeGoogle struct {
	identity identity.GoogleIdentity
	err      error
}

func (g *fakeGoogle) Exchange(context.Context, string, string, string) (identity.GoogleIdentity, error) {
	return g.identity, g.err
}

func newFixture(t *testing.T, limiter identity.RateLimiter, withGoogle bool) fixture {
	t.Helper()
	c := &clock{now: time.Date(2026, 10, 31, 9, 0, 0, 0, time.UTC)}
	repo := identitytest.NewRepository(c.Now)
	mailer := &identitytest.Mailer{}
	tokens, err := identity.NewTokenIssuer(testSecret, 15*time.Minute, c.Now)
	if err != nil {
		t.Fatal(err)
	}
	fx := fixture{repo: repo, mailer: mailer, clock: c, google: &fakeGoogle{}}
	deps := identity.Dependencies{
		Repository: repo,
		Tokens:     tokens,
		Mailer:     mailer,
		Limiter:    limiter,
		Now:        c.Now,
	}
	if withGoogle {
		deps.Google = fx.google
	}
	fx.service = identity.NewService(deps, identity.Config{
		AccessTokenTTL:   15 * time.Minute,
		RefreshTokenTTL:  30 * 24 * time.Hour,
		VerifyEmailTTL:   24 * time.Hour,
		ResetPasswordTTL: time.Hour,
		WebBaseURL:       "https://grupon.test",
		BrandName:        "Grupon",
	})
	return fx
}

func (fx fixture) register(t *testing.T) identity.Session {
	t.Helper()
	session, err := fx.service.Register(context.Background(), identity.RequestMeta{IP: "203.0.113.5", UserAgent: "test"}, identity.RegisterInput{
		Email: testEmail, Password: testPassword, Name: "Buyer", Locale: "en", Currency: "UZS",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	return session
}

// tokenFromLastMail extracts the single-use token from the most recent email.
func (fx fixture) tokenFromLastMail(t *testing.T) string {
	t.Helper()
	messages := fx.mailer.Messages()
	if len(messages) == 0 {
		t.Fatal("no email was sent")
	}
	match := tokenPattern.FindStringSubmatch(messages[len(messages)-1].Body)
	if match == nil {
		t.Fatal("email has no token link")
	}
	return match[1]
}

var meta = identity.RequestMeta{IP: "203.0.113.5", UserAgent: "test"}

func TestRegisterCreatesUserWithRoleAndSendsVerification(t *testing.T) {
	t.Parallel()
	fx := newFixture(t, nil, false)

	session := fx.register(t)

	if session.Account.Email != testEmail {
		t.Errorf("email = %q", session.Account.Email)
	}
	if !session.Account.HasRole(identity.RoleUser) {
		t.Errorf("roles = %v, want user", session.Account.Roles)
	}
	if session.AccessToken == "" || session.RefreshToken == "" {
		t.Fatal("register did not return tokens")
	}
	messages := fx.mailer.Messages()
	if len(messages) != 1 || messages[0].To != testEmail {
		t.Fatalf("messages = %+v, want one verification email", messages)
	}
	if got := fx.tokenFromLastMail(t); got == "" {
		t.Error("verification token missing")
	}
}

func TestRegisterDuplicateEmailIsRejected(t *testing.T) {
	t.Parallel()
	fx := newFixture(t, nil, false)
	fx.register(t)

	_, err := fx.service.Register(context.Background(), meta, identity.RegisterInput{
		Email: testEmail, Password: testPassword, Name: "Again", Locale: "en", Currency: "UZS",
	})
	if !errors.Is(err, identity.ErrEmailTaken) {
		t.Fatalf("error = %v, want ErrEmailTaken", err)
	}
}

func TestLoginErrorsAreIndistinguishable(t *testing.T) {
	t.Parallel()
	fx := newFixture(t, nil, false)
	fx.register(t)

	_, wrongPassword := fx.service.Login(context.Background(), meta, testEmail, "wrong-password-value")
	_, unknownEmail := fx.service.Login(context.Background(), meta, "nobody@example.test", testPassword)

	if !errors.Is(wrongPassword, identity.ErrInvalidCredentials) || !errors.Is(unknownEmail, identity.ErrInvalidCredentials) {
		t.Fatalf("errors = %v / %v, want ErrInvalidCredentials for both", wrongPassword, unknownEmail)
	}
	if wrongPassword.Error() != unknownEmail.Error() {
		t.Fatal("login errors differ and reveal account existence")
	}
}

func TestBlockedAccountCannotLogIn(t *testing.T) {
	t.Parallel()
	fx := newFixture(t, nil, false)
	session := fx.register(t)
	fx.repo.SetStatus(session.Account.ID, identity.StatusBlocked)

	_, err := fx.service.Login(context.Background(), meta, testEmail, testPassword)
	if !errors.Is(err, identity.ErrAccountBlocked) {
		t.Fatalf("error = %v, want ErrAccountBlocked", err)
	}
}

func TestRefreshRotatesAndReuseRevokesFamily(t *testing.T) {
	t.Parallel()
	fx := newFixture(t, nil, false)
	first := fx.register(t)

	second, err := fx.service.Refresh(context.Background(), meta, first.RefreshToken)
	if err != nil {
		t.Fatalf("first rotation error = %v", err)
	}
	if second.RefreshToken == first.RefreshToken {
		t.Fatal("refresh token was not rotated")
	}

	// Presenting the rotated token again is reuse: the whole family must die.
	_, err = fx.service.Refresh(context.Background(), meta, first.RefreshToken)
	if !errors.Is(err, identity.ErrRefreshTokenReused) {
		t.Fatalf("reuse error = %v, want ErrRefreshTokenReused", err)
	}
	if n := fx.repo.ActiveRefreshCount(first.Account.ID); n != 0 {
		t.Fatalf("active refresh tokens after reuse = %d, want 0", n)
	}
	_, err = fx.service.Refresh(context.Background(), meta, second.RefreshToken)
	if !errors.Is(err, identity.ErrRefreshTokenReused) {
		t.Fatalf("legitimate token after reuse error = %v, want ErrRefreshTokenReused", err)
	}
}

func TestRefreshExpiresAfterThirtyDays(t *testing.T) {
	t.Parallel()
	fx := newFixture(t, nil, false)
	session := fx.register(t)

	fx.clock.Advance(30*24*time.Hour + time.Minute)
	_, err := fx.service.Refresh(context.Background(), meta, session.RefreshToken)
	if !errors.Is(err, identity.ErrRefreshTokenInvalid) {
		t.Fatalf("error = %v, want ErrRefreshTokenInvalid", err)
	}
}

func TestLogoutRevokesRefreshToken(t *testing.T) {
	t.Parallel()
	fx := newFixture(t, nil, false)
	session := fx.register(t)

	if err := fx.service.Logout(context.Background(), session.RefreshToken); err != nil {
		t.Fatal(err)
	}
	_, err := fx.service.Refresh(context.Background(), meta, session.RefreshToken)
	if err == nil {
		t.Fatal("refresh after logout succeeded")
	}
}

func TestAccessTokenExpiresAndAuthenticateChecksStatus(t *testing.T) {
	t.Parallel()
	fx := newFixture(t, nil, false)
	session := fx.register(t)

	if _, err := fx.service.Authenticate(context.Background(), session.AccessToken); err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	fx.clock.Advance(16 * time.Minute)
	if _, err := fx.service.Authenticate(context.Background(), session.AccessToken); !errors.Is(err, identity.ErrUnauthenticated) {
		t.Fatalf("expired token error = %v, want ErrUnauthenticated", err)
	}
}

func TestVerifyEmailIsSingleUse(t *testing.T) {
	t.Parallel()
	fx := newFixture(t, nil, false)
	fx.register(t)
	token := fx.tokenFromLastMail(t)

	if err := fx.service.VerifyEmail(context.Background(), token); err != nil {
		t.Fatalf("first verify error = %v", err)
	}
	if err := fx.service.VerifyEmail(context.Background(), token); !errors.Is(err, identity.ErrInvalidToken) {
		t.Fatalf("second verify error = %v, want ErrInvalidToken", err)
	}
}

func TestVerifyEmailTokenExpires(t *testing.T) {
	t.Parallel()
	fx := newFixture(t, nil, false)
	fx.register(t)
	token := fx.tokenFromLastMail(t)

	fx.clock.Advance(25 * time.Hour)
	if err := fx.service.VerifyEmail(context.Background(), token); !errors.Is(err, identity.ErrInvalidToken) {
		t.Fatalf("error = %v, want ErrInvalidToken", err)
	}
}

func TestForgotPasswordIsSilentForUnknownEmail(t *testing.T) {
	t.Parallel()
	fx := newFixture(t, nil, false)

	if err := fx.service.ForgotPassword(context.Background(), meta, "nobody@example.test"); err != nil {
		t.Fatalf("ForgotPassword() error = %v, want nil", err)
	}
	if n := len(fx.mailer.Messages()); n != 0 {
		t.Fatalf("messages = %d, want 0 for unknown email", n)
	}
}

func TestResetPasswordFlowIsSingleUseAndRevokesSessions(t *testing.T) {
	t.Parallel()
	fx := newFixture(t, nil, false)
	session := fx.register(t)

	if err := fx.service.ForgotPassword(context.Background(), meta, testEmail); err != nil {
		t.Fatal(err)
	}
	token := fx.tokenFromLastMail(t)

	const newPassword = "brand-new-password-123"
	if err := fx.service.ResetPassword(context.Background(), meta, token, newPassword); err != nil {
		t.Fatalf("ResetPassword() error = %v", err)
	}
	if err := fx.service.ResetPassword(context.Background(), meta, token, newPassword); !errors.Is(err, identity.ErrInvalidToken) {
		t.Fatalf("second reset error = %v, want ErrInvalidToken", err)
	}
	if _, err := fx.service.Refresh(context.Background(), meta, session.RefreshToken); err == nil {
		t.Fatal("refresh token survived a password reset")
	}
	if _, err := fx.service.Login(context.Background(), meta, testEmail, testPassword); !errors.Is(err, identity.ErrInvalidCredentials) {
		t.Fatalf("old password error = %v, want ErrInvalidCredentials", err)
	}
	if _, err := fx.service.Login(context.Background(), meta, testEmail, newPassword); err != nil {
		t.Fatalf("new password login error = %v", err)
	}
}

func TestResetPasswordRejectsWeakPasswordWithoutBurningToken(t *testing.T) {
	t.Parallel()
	fx := newFixture(t, nil, false)
	fx.register(t)
	if err := fx.service.ForgotPassword(context.Background(), meta, testEmail); err != nil {
		t.Fatal(err)
	}
	token := fx.tokenFromLastMail(t)

	err := fx.service.ResetPassword(context.Background(), meta, token, "short")
	var validation *identity.ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("error = %v, want ValidationError", err)
	}
	if err := fx.service.ResetPassword(context.Background(), meta, token, "long-enough-password"); err != nil {
		t.Fatalf("token was consumed by an invalid password: %v", err)
	}
}

func TestChangePasswordKeepsCurrentSessionOnly(t *testing.T) {
	t.Parallel()
	fx := newFixture(t, nil, false)
	current := fx.register(t)
	other, err := fx.service.Login(context.Background(), meta, testEmail, testPassword)
	if err != nil {
		t.Fatal(err)
	}

	if err := fx.service.ChangePassword(context.Background(), current.Account.ID, testPassword, "another-password-123", current.RefreshToken); err != nil {
		t.Fatalf("ChangePassword() error = %v", err)
	}
	if _, err := fx.service.Refresh(context.Background(), meta, other.RefreshToken); err == nil {
		t.Error("other session survived password change")
	}
	if _, err := fx.service.Refresh(context.Background(), meta, current.RefreshToken); err != nil {
		t.Errorf("current session was revoked: %v", err)
	}
}

func TestChangePasswordRequiresCurrentPassword(t *testing.T) {
	t.Parallel()
	fx := newFixture(t, nil, false)
	session := fx.register(t)

	err := fx.service.ChangePassword(context.Background(), session.Account.ID, "not-the-password", "another-password-123", "")
	if !errors.Is(err, identity.ErrCurrentPasswordWrong) {
		t.Fatalf("error = %v, want ErrCurrentPasswordWrong", err)
	}
}

func TestDeleteAccountFreesEmailAndRevokesSessions(t *testing.T) {
	t.Parallel()
	fx := newFixture(t, nil, false)
	session := fx.register(t)

	if err := fx.service.DeleteAccount(context.Background(), session.Account.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := fx.service.Authenticate(context.Background(), session.AccessToken); !errors.Is(err, identity.ErrUnauthenticated) {
		t.Fatalf("deleted account authenticate error = %v", err)
	}
	fresh := fx.register(t)
	if fresh.Account.ID == session.Account.ID {
		t.Fatal("deleted account was reused")
	}
}

func TestRateLimitBlocksLoginAfterLimit(t *testing.T) {
	t.Parallel()
	fx := newFixture(t, &identitytest.Limiter{Max: 2}, false)
	fx.register(t)

	for i := 0; i < 2; i++ {
		_, _ = fx.service.Login(context.Background(), meta, testEmail, "wrong-password-value")
	}
	_, err := fx.service.Login(context.Background(), meta, testEmail, testPassword)
	if !errors.Is(err, identity.ErrRateLimited) {
		t.Fatalf("error = %v, want ErrRateLimited", err)
	}
}

type failingLimiter struct{}

func (failingLimiter) Allow(context.Context, string) (bool, error) {
	return false, errors.New("redis down")
}

func TestLimiterOutageFailsOpen(t *testing.T) {
	t.Parallel()
	fx := newFixture(t, failingLimiter{}, false)

	if _, err := fx.service.Register(context.Background(), meta, identity.RegisterInput{
		Email: testEmail, Password: testPassword, Name: "Buyer", Locale: "en", Currency: "UZS",
	}); err != nil {
		t.Fatalf("Register() with limiter outage error = %v, want success", err)
	}
}

func TestGoogleSignInCreatesUserAndLinksExistingVerifiedEmail(t *testing.T) {
	t.Parallel()
	fx := newFixture(t, nil, true)
	fx.google.identity = identity.GoogleIdentity{Subject: "google-sub-1", Email: testEmail, EmailVerified: true, Name: "Google Buyer"}

	created, err := fx.service.GoogleSignIn(context.Background(), meta, identity.GoogleSignInInput{Code: "c", CodeVerifier: "v", RedirectURI: "r"})
	if err != nil {
		t.Fatalf("GoogleSignIn() create error = %v", err)
	}
	again, err := fx.service.GoogleSignIn(context.Background(), meta, identity.GoogleSignInInput{Code: "c", CodeVerifier: "v", RedirectURI: "r"})
	if err != nil || again.Account.ID != created.Account.ID {
		t.Fatalf("second Google sign-in = %v, %v; want same account", again.Account.ID, err)
	}

	// An existing password account with the same verified email is linked, not duplicated.
	fx2 := newFixture(t, nil, true)
	existing := fx2.register(t)
	fx2.google.identity = identity.GoogleIdentity{Subject: "google-sub-2", Email: testEmail, EmailVerified: true}
	linked, err := fx2.service.GoogleSignIn(context.Background(), meta, identity.GoogleSignInInput{Code: "c", CodeVerifier: "v", RedirectURI: "r"})
	if err != nil {
		t.Fatalf("link error = %v", err)
	}
	if linked.Account.ID != existing.Account.ID {
		t.Fatal("Google sign-in created a duplicate account instead of linking")
	}
}

func TestGoogleSignInRejectsUnverifiedEmail(t *testing.T) {
	t.Parallel()
	fx := newFixture(t, nil, true)
	fx.google.identity = identity.GoogleIdentity{Subject: "sub", Email: testEmail, EmailVerified: false}

	_, err := fx.service.GoogleSignIn(context.Background(), meta, identity.GoogleSignInInput{Code: "c", CodeVerifier: "v", RedirectURI: "r"})
	if !errors.Is(err, identity.ErrGoogleEmailUnverified) {
		t.Fatalf("error = %v, want ErrGoogleEmailUnverified", err)
	}
}

func TestGoogleSignInNotConfigured(t *testing.T) {
	t.Parallel()
	fx := newFixture(t, nil, false)

	_, err := fx.service.GoogleSignIn(context.Background(), meta, identity.GoogleSignInInput{})
	if !errors.Is(err, identity.ErrGoogleNotConfigured) {
		t.Fatalf("error = %v, want ErrGoogleNotConfigured", err)
	}
}
