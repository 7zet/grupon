package store_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/grupon/grupon/services/api/internal/domain/identity"
	"github.com/grupon/grupon/services/api/internal/domain/identity/identitytest"
	"github.com/grupon/grupon/services/api/internal/store"
)

var identityTokenPattern = regexp.MustCompile(`token=([A-Za-z0-9_-]+)`)

// newIdentityFixture migrates TEST_DATABASE_URL and returns a service backed by PostgreSQL. Every
// test uses fresh email addresses instead of truncating shared tables, so seeded data survives.
func newIdentityFixture(t *testing.T) (*identity.Service, *identitytest.Mailer, string) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()

	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	_, file, _, _ := runtime.Caller(0)
	if err := goose.UpContext(ctx, db, filepath.Join(filepath.Dir(file), "..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}

	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	tokens, err := identity.NewTokenIssuer("integration-test-secret-with-32-bytes!", 15*time.Minute, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	mailer := &identitytest.Mailer{}
	service := identity.NewService(identity.Dependencies{
		Repository: store.NewIdentityRepository(pool),
		Tokens:     tokens,
		Mailer:     mailer,
		Now:        time.Now,
	}, identity.Config{
		AccessTokenTTL:   15 * time.Minute,
		RefreshTokenTTL:  30 * 24 * time.Hour,
		VerifyEmailTTL:   24 * time.Hour,
		ResetPasswordTTL: time.Hour,
		WebBaseURL:       "https://grupon.test",
		BrandName:        "Grupon",
	})
	email := fmt.Sprintf("it-%s@example.test", uuid.NewString())
	return service, mailer, email
}

func lastToken(t *testing.T, mailer *identitytest.Mailer) string {
	t.Helper()
	messages := mailer.Messages()
	match := identityTokenPattern.FindStringSubmatch(messages[len(messages)-1].Body)
	if match == nil {
		t.Fatal("email has no token")
	}
	return match[1]
}

var integrationMeta = identity.RequestMeta{IP: "198.51.100.7", UserAgent: "integration"}

func TestIdentityLifecycleAgainstPostgres(t *testing.T) {
	service, mailer, email := newIdentityFixture(t)
	ctx := context.Background()
	const password = "integration-password-1"

	session, err := service.Register(ctx, integrationMeta, identity.RegisterInput{
		Email: email, Password: password, Name: "Integration", Locale: "en", Currency: "UZS",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if !session.Account.HasRole(identity.RoleUser) {
		t.Fatalf("roles = %v, want user from the database", session.Account.Roles)
	}

	if _, err := service.Register(ctx, integrationMeta, identity.RegisterInput{
		Email: email, Password: password, Name: "Again", Locale: "en", Currency: "UZS",
	}); !errors.Is(err, identity.ErrEmailTaken) {
		t.Fatalf("duplicate register error = %v, want ErrEmailTaken", err)
	}

	if err := service.VerifyEmail(ctx, lastToken(t, mailer)); err != nil {
		t.Fatalf("VerifyEmail() error = %v", err)
	}

	login, err := service.Login(ctx, integrationMeta, email, password)
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	rotated, err := service.Refresh(ctx, integrationMeta, login.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if _, err := service.Refresh(ctx, integrationMeta, login.RefreshToken); !errors.Is(err, identity.ErrRefreshTokenReused) {
		t.Fatalf("reuse error = %v, want ErrRefreshTokenReused", err)
	}
	if _, err := service.Refresh(ctx, integrationMeta, rotated.RefreshToken); !errors.Is(err, identity.ErrRefreshTokenReused) {
		t.Fatalf("family should be revoked, got %v", err)
	}

	account, err := service.Authenticate(ctx, login.AccessToken)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if account.EmailVerifiedAt == nil {
		t.Error("email_verified_at not set after verification")
	}

	if err := service.ForgotPassword(ctx, integrationMeta, email); err != nil {
		t.Fatalf("ForgotPassword() error = %v", err)
	}
	resetToken := lastToken(t, mailer)
	const newPassword = "integration-password-2"
	if err := service.ResetPassword(ctx, integrationMeta, resetToken, newPassword); err != nil {
		t.Fatalf("ResetPassword() error = %v", err)
	}
	if err := service.ResetPassword(ctx, integrationMeta, resetToken, newPassword); !errors.Is(err, identity.ErrInvalidToken) {
		t.Fatalf("reset token reuse error = %v, want ErrInvalidToken", err)
	}
	if _, err := service.Login(ctx, integrationMeta, email, password); !errors.Is(err, identity.ErrInvalidCredentials) {
		t.Fatalf("old password error = %v", err)
	}
	if _, err := service.Login(ctx, integrationMeta, email, newPassword); err != nil {
		t.Fatalf("new password login error = %v", err)
	}

	if err := service.DeleteAccount(ctx, account.ID); err != nil {
		t.Fatalf("DeleteAccount() error = %v", err)
	}
	if _, err := service.Authenticate(ctx, login.AccessToken); !errors.Is(err, identity.ErrUnauthenticated) {
		t.Fatalf("deleted account error = %v, want ErrUnauthenticated", err)
	}
}

func TestConcurrentRefreshAllowsExactlyOneRotation(t *testing.T) {
	service, _, email := newIdentityFixture(t)
	ctx := context.Background()

	session, err := service.Register(ctx, integrationMeta, identity.RegisterInput{
		Email: email, Password: "integration-password-1", Name: "Race", Locale: "uz", Currency: "UZS",
	})
	if err != nil {
		t.Fatal(err)
	}

	const racers = 8
	var wg sync.WaitGroup
	results := make(chan error, racers)
	for i := 0; i < racers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := service.Refresh(ctx, integrationMeta, session.RefreshToken)
			results <- err
		}()
	}
	wg.Wait()
	close(results)

	successes := 0
	for err := range results {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("successful rotations = %d, want exactly 1", successes)
	}
}

func TestGoogleSubjectRoundTripsThroughRepository(t *testing.T) {
	_, _, _ = newIdentityFixture(t)
	url := os.Getenv("TEST_DATABASE_URL")
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	repo := store.NewIdentityRepository(pool)
	sub := "google-" + uuid.NewString()
	first, err := repo.CreateUser(context.Background(), identity.NewUser{
		Email: fmt.Sprintf("g-%s@example.test", uuid.NewString()), GoogleSub: &sub, Name: "G", Locale: "en", Currency: "UZS", EmailVerified: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.GoogleSub == nil || *first.GoogleSub != sub {
		t.Fatal("google_sub not stored")
	}
	found, err := repo.GetUserByGoogleSub(context.Background(), sub)
	if err != nil || found.ID != first.ID {
		t.Fatalf("GetUserByGoogleSub() = %v, %v", found.ID, err)
	}
}
