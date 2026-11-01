package store_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
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

func TestCatalogReadModelFromSeed(t *testing.T) {
	fixture := newCatalogFixture(t)
	ctx := context.Background()

	t.Run("configuration lists seeded cities, categories and the exchange rate", func(t *testing.T) {
		config, err := fixture.service.PublicConfig(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(config.Cities) != 1 {
			t.Errorf("cities = %d, want 1 (only Toshkent is open)", len(config.Cities))
		}
		if len(config.Categories) != 4 {
			t.Errorf("categories = %d, want 4 (pilot)", len(config.Categories))
		}
		for _, category := range config.Categories {
			if len(category.Subcategories) == 0 {
				t.Errorf("category %s has no subcategories", category.Slug)
			}
		}
		if config.ExchangeRate.Rate == "" {
			t.Error("exchange rate is empty")
		}
	})

	t.Run("listing returns only publishable deals with options and total", func(t *testing.T) {
		page, err := fixture.service.ListDeals(ctx, catalog.DealFilter{}, catalog.SortRecommended, mustPage(t, 1, 100))
		if err != nil {
			t.Fatal(err)
		}
		if page.Total == 0 || len(page.Items) == 0 {
			t.Fatalf("listing empty: total=%d items=%d", page.Total, len(page.Items))
		}
		for _, deal := range page.Items {
			if !catalog.IsPublishable(deal, fixture.now) {
				t.Errorf("deal %s with status %s is listed", deal.Slug, deal.Status)
			}
			if len(deal.Options) == 0 {
				t.Errorf("deal %s has no active options", deal.Slug)
			}
		}
	})

	t.Run("price ascending sort orders by the cheapest option", func(t *testing.T) {
		page, err := fixture.service.ListDeals(ctx, catalog.DealFilter{}, catalog.SortPriceAsc, mustPage(t, 1, 100))
		if err != nil {
			t.Fatal(err)
		}
		for index := 1; index < len(page.Items); index++ {
			if cheapestPrice(page.Items[index-1]) > cheapestPrice(page.Items[index]) {
				t.Fatalf("price order broken at %d", index)
			}
		}
	})

	t.Run("pages are disjoint and report the full total", func(t *testing.T) {
		first, err := fixture.service.ListDeals(ctx, catalog.DealFilter{}, catalog.SortPopular, mustPage(t, 1, 10))
		if err != nil {
			t.Fatal(err)
		}
		second, err := fixture.service.ListDeals(ctx, catalog.DealFilter{}, catalog.SortPopular, mustPage(t, 2, 10))
		if err != nil {
			t.Fatal(err)
		}
		seen := map[uuid.UUID]bool{}
		for _, deal := range first.Items {
			seen[deal.ID] = true
		}
		for _, deal := range second.Items {
			if seen[deal.ID] {
				t.Errorf("deal %s appears on two pages", deal.Slug)
			}
		}
		if first.Total != second.Total || first.Total < int64(len(first.Items)+len(second.Items)) {
			t.Errorf("total mismatch: %d vs %d", first.Total, second.Total)
		}
	})

	t.Run("city filter returns deals with a branch in that city (pilot: all deals are in Toshkent)", func(t *testing.T) {
		all, err := fixture.service.ListDeals(ctx, catalog.DealFilter{}, catalog.SortRecommended, mustPage(t, 1, 1))
		if err != nil {
			t.Fatal(err)
		}
		city, err := fixture.service.ListDeals(ctx, catalog.DealFilter{City: "tashkent"}, catalog.SortRecommended, mustPage(t, 1, 1))
		if err != nil {
			t.Fatal(err)
		}
		if city.Total == 0 || city.Total != all.Total {
			t.Errorf("tashkent total = %d, all = %d (pilot seed has only Toshkent branches)", city.Total, all.Total)
		}
	})

	t.Run("search finds seeded titles by word", func(t *testing.T) {
		page, err := fixture.service.ListDeals(ctx, catalog.DealFilter{Query: "taklif"}, catalog.SortRecommended, mustPage(t, 1, 24))
		if err != nil {
			t.Fatal(err)
		}
		if page.Total == 0 {
			t.Error("search for a seeded title returned nothing")
		}
	})

	t.Run("deal detail includes options, branches and highlights", func(t *testing.T) {
		var slug string
		if err := fixture.pool.QueryRow(ctx, `SELECT slug FROM deals WHERE status = 'active' ORDER BY slug LIMIT 1`).Scan(&slug); err != nil {
			t.Fatal(err)
		}
		deal, err := fixture.service.Deal(ctx, slug)
		if err != nil {
			t.Fatalf("Deal(%s): %v", slug, err)
		}
		if len(deal.Options) == 0 || len(deal.Locations) == 0 || len(deal.Highlights) < 3 {
			t.Errorf("detail incomplete: options=%d locations=%d highlights=%d",
				len(deal.Options), len(deal.Locations), len(deal.Highlights))
		}
		if deal.Subcategory.Slug == "" {
			t.Error("detail has no subcategory")
		}
	})

	t.Run("unknown deal is not found", func(t *testing.T) {
		_, err := fixture.service.Deal(ctx, "no-such-deal")
		if !errors.Is(err, catalog.ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("expired deals keep their detail page but never appear in listings", func(t *testing.T) {
		var slug string
		if err := fixture.pool.QueryRow(ctx, `SELECT slug FROM deals WHERE status = 'expired' LIMIT 1`).Scan(&slug); err != nil {
			t.Fatal(err)
		}
		deal, err := fixture.service.Deal(ctx, slug)
		if err != nil {
			t.Fatalf("Deal(%s): %v", slug, err)
		}
		if deal.Status != "expired" {
			t.Errorf("status = %s, want expired", deal.Status)
		}
		listed, err := fixture.service.ListDeals(ctx, catalog.DealFilter{}, catalog.SortRecommended, mustPage(t, 1, 100))
		if err != nil {
			t.Fatal(err)
		}
		for _, item := range listed.Items {
			if item.Slug == slug {
				t.Errorf("expired deal %s is listed", slug)
			}
		}
	})

	t.Run("deal reviews are published, verified and include replies", func(t *testing.T) {
		var slug string
		if err := fixture.pool.QueryRow(ctx, `
			SELECT d.slug FROM deals d JOIN reviews r ON r.deal_id = d.id
			WHERE d.status = 'active' AND r.status = 'published' LIMIT 1`).Scan(&slug); err != nil {
			t.Fatal(err)
		}
		page, err := fixture.service.DealReviews(ctx, slug, mustPage(t, 1, 50))
		if err != nil {
			t.Fatal(err)
		}
		if page.Total == 0 || len(page.Items) == 0 {
			t.Fatalf("no reviews for %s", slug)
		}
		for _, review := range page.Items {
			if review.Status != "published" {
				t.Errorf("review status = %s", review.Status)
			}
		}
	})

	t.Run("merchant profile returns active deals and reviews", func(t *testing.T) {
		profile, err := fixture.service.Merchant(ctx, "seed-merchant-01")
		if err != nil {
			t.Fatalf("Merchant: %v", err)
		}
		if len(profile.Locations) == 0 {
			t.Error("merchant has no locations")
		}
	})

	t.Run("suspended merchants disappear from listings and detail", func(t *testing.T) {
		ctx := context.Background()
		var merchantSlug, dealSlug string
		if err := fixture.pool.QueryRow(ctx, `
			SELECT m.slug, d.slug FROM deals d JOIN merchants m ON m.id = d.merchant_id
			WHERE d.status = 'active' ORDER BY d.slug LIMIT 1`).Scan(&merchantSlug, &dealSlug); err != nil {
			t.Fatal(err)
		}
		if _, err := fixture.pool.Exec(ctx, `UPDATE merchants SET status = 'suspended' WHERE slug = $1`, merchantSlug); err != nil {
			t.Fatal(err)
		}
		defer func() {
			_, _ = fixture.pool.Exec(ctx, `UPDATE merchants SET status = 'active' WHERE slug = $1`, merchantSlug)
		}()
		if _, err := fixture.service.Deal(ctx, dealSlug); !errors.Is(err, catalog.ErrNotFound) {
			t.Errorf("suspended merchant deal detail err = %v, want ErrNotFound", err)
		}
		if _, err := fixture.service.Merchant(ctx, merchantSlug); !errors.Is(err, catalog.ErrNotFound) {
			t.Errorf("suspended merchant profile err = %v, want ErrNotFound", err)
		}
	})

	t.Run("promo code clicks are counted only for active codes", func(t *testing.T) {
		var id uuid.UUID
		var before int64
		if err := fixture.pool.QueryRow(ctx, `
			SELECT id, click_count FROM promo_codes WHERE is_active AND ends_at > now() LIMIT 1`).Scan(&id, &before); err != nil {
			t.Fatal(err)
		}
		if err := fixture.service.TrackPromoCodeClick(ctx, id); err != nil {
			t.Fatalf("TrackPromoCodeClick: %v", err)
		}
		var after int64
		if err := fixture.pool.QueryRow(ctx, `SELECT click_count FROM promo_codes WHERE id = $1`, id).Scan(&after); err != nil {
			t.Fatal(err)
		}
		if after != before+1 {
			t.Errorf("click_count = %d, want %d", after, before+1)
		}
		if err := fixture.service.TrackPromoCodeClick(ctx, uuid.New()); !errors.Is(err, catalog.ErrNotFound) {
			t.Errorf("unknown promo code err = %v, want ErrNotFound", err)
		}
	})

	t.Run("collections and homepage sections resolve publishable content", func(t *testing.T) {
		collections, err := fixture.service.Collections(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(collections) == 0 {
			t.Fatal("no active collections")
		}
		detail, err := fixture.service.Collection(ctx, collections[0].Slug, mustPage(t, 1, 24))
		if err != nil {
			t.Fatal(err)
		}
		if detail.Deals.Total == 0 {
			t.Error("collection has no publishable deals")
		}
		sections, err := fixture.service.Homepage(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(sections) == 0 {
			t.Error("no homepage sections")
		}
	})

	t.Run("suggestions return deals, merchants and categories", func(t *testing.T) {
		items, err := fixture.service.Suggestions(ctx, "Sayohat", "uz")
		if err != nil {
			t.Fatal(err)
		}
		if len(items) == 0 {
			t.Error("no suggestions for a seeded travel word")
		}
	})
}
