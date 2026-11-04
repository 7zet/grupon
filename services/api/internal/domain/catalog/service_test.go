package catalog

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

// fakeRepository serves canned deals and records how often the summary loader runs.
type fakeRepository struct {
	Repository
	summaries    []DealSummary
	summaryCalls int
	searchedIDs  []uuid.UUID
}

func (r *fakeRepository) SearchDealIDs(_ context.Context, _ DealFilter, _ Sort, _ Page) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0, len(r.summaries))
	for _, deal := range r.summaries {
		ids = append(ids, deal.ID)
	}
	return ids, nil
}

func (r *fakeRepository) CountDeals(_ context.Context, _ DealFilter) (int64, error) {
	return int64(len(r.summaries)), nil
}

func (r *fakeRepository) DealSummaries(_ context.Context, ids []uuid.UUID) ([]DealSummary, error) {
	r.summaryCalls++
	result := []DealSummary{}
	for _, id := range ids {
		for _, deal := range r.summaries {
			if deal.ID == id {
				result = append(result, deal)
			}
		}
	}
	return result, nil
}

// memoryCache records entry presence only; that is enough to observe hits and invalidation.
type memoryCache struct {
	entries map[string]struct{}
	hits    int
}

func (c *memoryCache) Get(_ context.Context, key string, _ any) (bool, error) {
	if _, ok := c.entries[key]; ok {
		c.hits++
		return true, nil
	}
	return false, nil
}

func (c *memoryCache) Set(_ context.Context, key string, _ any, _ time.Duration) error {
	c.entries[key] = struct{}{}
	return nil
}

func (c *memoryCache) Invalidate(_ context.Context) error {
	c.entries = map[string]struct{}{}
	return nil
}

func TestParseSortAcceptsContractValuesOnly(t *testing.T) {
	t.Parallel()

	if got, err := ParseSort(""); err != nil || got != SortRecommended {
		t.Fatalf("ParseSort(\"\") = %q, %v; want recommended", got, err)
	}
	if got, err := ParseSort("price_asc"); err != nil || got != SortPriceAsc {
		t.Fatalf("ParseSort(price_asc) = %q, %v", got, err)
	}
	if _, err := ParseSort("price-asc"); err == nil {
		t.Fatal("ParseSort accepted a non-contract value")
	}
}

func TestNewPageAppliesDefaultsAndBounds(t *testing.T) {
	t.Parallel()

	page, err := NewPage(nil, nil)
	if err != nil || page.Number != 1 || page.Size != defaultPageSize {
		t.Fatalf("defaults = %+v, %v", page, err)
	}
	zero, big := 0, maxPageSize+1
	if _, err := NewPage(&zero, nil); err == nil {
		t.Error("page 0 accepted")
	}
	if _, err := NewPage(nil, &big); err == nil {
		t.Error("oversized page accepted")
	}
	two, size := 2, 10
	page, _ = NewPage(&two, &size)
	if page.Offset() != 10 {
		t.Errorf("Offset() = %d, want 10", page.Offset())
	}
}

func TestTextResolveFallsBackInOrder(t *testing.T) {
	t.Parallel()

	text := Text{Uz: "", Ru: "Привет", En: "Hello"}
	if got := text.Resolve("en"); got != "Hello" {
		t.Errorf("Resolve(en) = %q", got)
	}
	if got := text.Resolve("uz"); got != "Привет" {
		t.Errorf("Resolve(uz) fallback = %q, want ru", got)
	}
	if got := (Text{}).Resolve("ru"); got != "" {
		t.Errorf("empty text = %q", got)
	}
}

func TestSearchQueryTrimsAndRejectsLongInput(t *testing.T) {
	t.Parallel()

	if got, err := SearchQuery("  spa  "); err != nil || got != "spa" {
		t.Fatalf("SearchQuery = %q, %v", got, err)
	}
	long := make([]rune, maxSearchLength+1)
	for index := range long {
		long[index] = 'a'
	}
	if _, err := SearchQuery(string(long)); err == nil {
		t.Error("over-long search accepted")
	}
}

func TestListDealsDropsUnpublishedAndKeepsRepositoryOrder(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	live := DealSummary{ID: uuid.New(), Slug: "live", Status: "active", EndsAt: now.Add(time.Hour)}
	soldOut := DealSummary{ID: uuid.New(), Slug: "sold", Status: "active", EndsAt: now.Add(time.Hour)} // sold out is stock, not a state
	expired := DealSummary{ID: uuid.New(), Slug: "old", Status: "active", EndsAt: now.Add(-time.Hour)}
	draft := DealSummary{ID: uuid.New(), Slug: "draft", Status: "draft", EndsAt: now.Add(time.Hour)}
	repo := &fakeRepository{summaries: []DealSummary{live, expired, draft, soldOut}}
	service := NewService(repo, nil, func() time.Time { return now })

	page, err := service.ListDeals(context.Background(), DealFilter{}, SortRecommended, Page{Number: 1, Size: 24})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 2 || page.Items[0].Slug != "live" || page.Items[1].Slug != "sold" {
		t.Fatalf("items = %+v, want live then sold", page.Items)
	}
}

func TestCachedReadsSkipRepositoryUntilInvalidated(t *testing.T) {
	t.Parallel()

	now := time.Now()
	deal := DealSummary{ID: uuid.New(), Slug: "a", Status: "active", EndsAt: now.Add(time.Hour)}
	repo := &fakeRepository{summaries: []DealSummary{deal}}
	cache := &memoryCache{entries: map[string]struct{}{}}
	service := NewService(repo, cache, func() time.Time { return now })
	ctx := context.Background()

	if _, err := service.ListDeals(ctx, DealFilter{}, SortRecommended, Page{Number: 1, Size: 24}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ListDeals(ctx, DealFilter{}, SortRecommended, Page{Number: 1, Size: 24}); err != nil {
		t.Fatal(err)
	}
	if cache.hits != 1 {
		t.Fatalf("cache hits = %d, want 1 on the second identical read", cache.hits)
	}
	if err := service.Invalidate(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ListDeals(ctx, DealFilter{}, SortRecommended, Page{Number: 1, Size: 24}); err != nil {
		t.Fatal(err)
	}
	if cache.hits != 1 {
		t.Fatalf("read after invalidation was served from cache")
	}
}

func TestUnknownDealSurfacesNotFound(t *testing.T) {
	t.Parallel()

	service := NewService(&notFoundRepository{}, nil, time.Now)
	_, err := service.Deal(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

type notFoundRepository struct{ Repository }

func (notFoundRepository) DealBySlug(context.Context, string) (Deal, error) {
	return Deal{}, ErrNotFound
}
