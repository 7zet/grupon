package store_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/grupon/grupon/services/api/internal/domain/catalog"
	"github.com/grupon/grupon/services/api/internal/seed"
	"github.com/grupon/grupon/services/api/internal/store"
)

// catalogFixture is a freshly seeded database with the public catalog service wired to it.
type catalogFixture struct {
	pool    *pgxpool.Pool
	service *catalog.Service
	now     time.Time
}

// newCatalogFixture seeds a database and returns the catalog service. TEST_DATABASE_URL must point
// at a disposable database because the seed truncates its tables; otherwise a container is used.
func newCatalogFixture(t *testing.T) catalogFixture {
	t.Helper()
	ctx := context.Background()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		container, err := postgres.Run(ctx, "postgres:18.6-alpine",
			postgres.WithDatabase("app"), postgres.WithUsername("app"), postgres.WithPassword("app"),
			testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(60*time.Second)),
		)
		if err != nil {
			t.Skipf("Docker is unavailable and TEST_DATABASE_URL is not set: %v", err)
		}
		t.Cleanup(func() { _ = container.Terminate(ctx) })
		url, err = container.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := seed.Run(ctx, url); err != nil {
		t.Fatalf("seed database: %v", err)
	}
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	now := time.Now()
	repo := store.NewCatalogRepository(pool)
	return catalogFixture{
		pool:    pool,
		service: catalog.NewService(repo, nil, func() time.Time { return now }),
		now:     now,
	}
}

func mustPage(t *testing.T, number, size int) catalog.Page {
	t.Helper()
	page, err := catalog.NewPage(&number, &size)
	if err != nil {
		t.Fatal(err)
	}
	return page
}

func cheapestPrice(deal catalog.DealSummary) int64 {
	cheapest := int64(-1)
	for _, option := range deal.Options {
		if cheapest < 0 || option.DealPrice.Amount < cheapest {
			cheapest = option.DealPrice.Amount
		}
	}
	return cheapest
}
