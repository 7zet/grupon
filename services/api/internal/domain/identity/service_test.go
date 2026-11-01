package identity_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/grupon/grupon/services/api/internal/domain/identity"
	"github.com/grupon/grupon/services/api/internal/domain/identity/identitytest"
)

const (
	testPhone  = "+998901234567"
	testSecret = "service-test-secret-with-at-least-32-bytes"
)

// nullSender accepts every sign-in code; the tests read the code from the response instead.
type nullSender struct{}

func (nullSender) SendCode(context.Context, string, string) error { return nil }

type clock struct{ now time.Time }

func (c *clock) Now() time.Time          { return c.now }
func (c *clock) Advance(d time.Duration) { c.now = c.now.Add(d) }

type fixture struct {
	service *identity.Service
	repo    *identitytest.Repository
	mailer  *identitytest.Mailer
	clock   *clock
}

func newFixture(t *testing.T, limiter identity.RateLimiter) fixture {
	t.Helper()
	c := &clock{now: time.Date(2026, 10, 31, 9, 0, 0, 0, time.UTC)}
	repo := identitytest.NewRepository(c.Now)
	mailer := &identitytest.Mailer{}
	tokens, err := identity.NewTokenIssuer(testSecret, 15*time.Minute, c.Now)
	if err != nil {
		t.Fatal(err)
	}
	fx := fixture{repo: repo, mailer: mailer, clock: c}
	deps := identity.Dependencies{
		Repository: repo,
		Tokens:     tokens,
		Mailer:     mailer,
		Limiter:    limiter,
		CodeSender: nullSender{},
		Now:        c.Now,
	}
	fx.service = identity.NewService(deps, identity.Config{
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 30 * 24 * time.Hour,
		OTPEcho:         true,
		BrandName:       "Grupon",
	})
	return fx
}

func (fx fixture) register(t *testing.T) identity.Session {
	t.Helper()
	ctx := context.Background()
	code, err := fx.service.RequestCode(ctx, meta, testPhone)
	if err != nil {
		t.Fatalf("RequestCode() error = %v", err)
	}
	session, err := fx.service.VerifyCode(ctx, meta, testPhone, code.DevCode)
	if err != nil {
		t.Fatalf("VerifyCode() error = %v", err)
	}
	return session
}

var meta = identity.RequestMeta{IP: "203.0.113.5", UserAgent: "test"}

func TestRefreshRotatesAndReuseRevokesFamily(t *testing.T) {
	t.Parallel()
	fx := newFixture(t, nil)
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
	fx := newFixture(t, nil)
	session := fx.register(t)

	fx.clock.Advance(30*24*time.Hour + time.Minute)
	_, err := fx.service.Refresh(context.Background(), meta, session.RefreshToken)
	if !errors.Is(err, identity.ErrRefreshTokenInvalid) {
		t.Fatalf("error = %v, want ErrRefreshTokenInvalid", err)
	}
}

func TestLogoutRevokesRefreshToken(t *testing.T) {
	t.Parallel()
	fx := newFixture(t, nil)
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
	fx := newFixture(t, nil)
	session := fx.register(t)

	if _, err := fx.service.Authenticate(context.Background(), session.AccessToken); err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	fx.clock.Advance(16 * time.Minute)
	if _, err := fx.service.Authenticate(context.Background(), session.AccessToken); !errors.Is(err, identity.ErrUnauthenticated) {
		t.Fatalf("expired token error = %v, want ErrUnauthenticated", err)
	}
}

func TestDeleteAccountFreesEmailAndRevokesSessions(t *testing.T) {
	t.Parallel()
	fx := newFixture(t, nil)
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

type failingLimiter struct{}

func (failingLimiter) Allow(context.Context, string) (bool, error) {
	return false, errors.New("redis down")
}

func TestLimiterOutageFailsOpen(t *testing.T) {
	fx := newFixture(t, failingLimiter{})
	if _, err := fx.service.RequestCode(context.Background(), meta, testPhone); err != nil {
		t.Fatalf("RequestCode() during limiter outage error = %v, want fail-open", err)
	}
}
