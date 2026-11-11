package store_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"database/sql"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/dealo/dealo/services/api/internal/domain/identity"
	"github.com/dealo/dealo/services/api/internal/store"
)

// phoneSender accepts every sign-in code; tests read the code from the response (development echo).
type phoneSender struct{}

func (phoneSender) SendCode(context.Context, string, string) error { return nil }

// newIdentityFixture migrates TEST_DATABASE_URL and returns a service backed by PostgreSQL. Every
// test uses a fresh phone number instead of truncating shared tables, so seeded data survives.
func newIdentityFixture(t *testing.T) (*identity.Service, string) {
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
	service := identity.NewService(identity.Dependencies{
		Repository: store.NewIdentityRepository(pool),
		Tokens:     tokens,
		CodeSender: phoneSender{},
		Now:        time.Now,
	}, identity.Config{
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 30 * 24 * time.Hour,
		OTPEcho:         true,
		BrandName:       "Dealo",
	})
	return service, fmt.Sprintf("+99890%07d", time.Now().UnixNano()%10_000_000)
}

// signIn requests a code for the phone and redeems it.
func signIn(t *testing.T, service *identity.Service, phone string) identity.Session {
	t.Helper()
	ctx := context.Background()
	code, err := service.RequestCode(ctx, integrationMeta, phone)
	if err != nil {
		t.Fatalf("RequestCode() error = %v", err)
	}
	session, err := service.VerifyCode(ctx, integrationMeta, phone, code.DevCode)
	if err != nil {
		t.Fatalf("VerifyCode() error = %v", err)
	}
	return session
}

var integrationMeta = identity.RequestMeta{IP: "198.51.100.7", UserAgent: "integration"}

func TestIdentityLifecycleAgainstPostgres(t *testing.T) {
	service, phone := newIdentityFixture(t)
	ctx := context.Background()

	session := signIn(t, service, phone)
	if !session.Account.HasRole(identity.RoleUser) {
		t.Fatalf("roles = %v, want user from the database", session.Account.Roles)
	}

	rotated, err := service.Refresh(ctx, integrationMeta, session.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if _, err := service.Refresh(ctx, integrationMeta, session.RefreshToken); !errors.Is(err, identity.ErrRefreshTokenReused) {
		t.Fatalf("reuse error = %v, want ErrRefreshTokenReused", err)
	}
	if _, err := service.Refresh(ctx, integrationMeta, rotated.RefreshToken); !errors.Is(err, identity.ErrRefreshTokenReused) {
		t.Fatalf("family should be revoked, got %v", err)
	}

	if _, err := service.Authenticate(ctx, rotated.AccessToken); err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
}

func TestConcurrentRefreshAllowsExactlyOneRotation(t *testing.T) {
	service, phone := newIdentityFixture(t)
	ctx := context.Background()
	session := signIn(t, service, phone)

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
