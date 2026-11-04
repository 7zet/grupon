package store_test

import (
	"context"
	"testing"

	"github.com/grupon/grupon/services/api/internal/store"
)

// TestDealSweepsAreIdempotent proves that a second run of the lifecycle sweeps changes no rows, so
// overlapping or repeated worker ticks cannot corrupt deal state.
func TestDealSweepsAreIdempotent(t *testing.T) {
	fixture := newCatalogFixture(t)
	ctx := context.Background()
	jobs := store.NewDealJobs(fixture.pool)

	if _, err := jobs.RunDealSweeps(ctx); err != nil {
		t.Fatalf("first run: %v", err)
	}
	second, err := jobs.RunDealSweeps(ctx)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if second != (store.DealSweepResult{}) {
		t.Fatalf("second run changed rows: %+v", second)
	}
}

// TestEndedDealsLeaveTheCatalog checks that the expiry sweep removes ended deals from listings.
func TestEndedDealsLeaveTheCatalog(t *testing.T) {
	fixture := newCatalogFixture(t)
	ctx := context.Background()

	var slug string
	if err := fixture.pool.QueryRow(ctx,
		`SELECT slug FROM deals WHERE status = 'active' ORDER BY slug LIMIT 1`).Scan(&slug); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.pool.Exec(ctx,
		`UPDATE deals SET ends_at = now() - interval '1 hour' WHERE slug = $1`, slug); err != nil {
		t.Fatal(err)
	}
	if _, err := store.NewDealJobs(fixture.pool).RunDealSweeps(ctx); err != nil {
		t.Fatal(err)
	}
	var status string
	if err := fixture.pool.QueryRow(ctx, `SELECT status FROM deals WHERE slug = $1`, slug).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "ended" {
		t.Fatalf("status = %s, want ended", status)
	}
}
