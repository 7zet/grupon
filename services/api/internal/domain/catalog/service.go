package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	defaultPageSize = 24
	maxPageSize     = 100
	maxSuggestions  = 10
	maxSearchLength = 200
	relatedLimit    = 8
	merchantReviews = 20
	// endingSoonWindow matches the product rule that urgency is shown only inside 72 hours.
	endingSoonWindow = 72 * time.Hour
	cacheTTL         = 60 * time.Second
	cacheTTLStatic   = 10 * time.Minute
)

// Sort is one of the listing orderings allowed by the public contract.
type Sort string

const (
	SortRecommended     Sort = "recommended"
	SortPopular         Sort = "popular"
	SortNewest          Sort = "newest"
	SortPriceAsc        Sort = "price_asc"
	SortPriceDesc       Sort = "price_desc"
	SortBiggestDiscount Sort = "biggest_discount"
	SortEndingSoon      Sort = "ending_soon"
)

// ParseSort validates a client-supplied sort value. An empty value means recommended.
func ParseSort(raw string) (Sort, error) {
	switch Sort(raw) {
	case "":
		return SortRecommended, nil
	case SortRecommended, SortPopular, SortNewest, SortPriceAsc, SortPriceDesc,
		SortBiggestDiscount, SortEndingSoon:
		return Sort(raw), nil
	default:
		return "", invalid("sort", "invalid_sort")
	}
}

// Page is a validated one-based page request.
type Page struct {
	Number int
	Size   int
}

// NewPage validates paging input and applies defaults for absent values.
func NewPage(number, size *int) (Page, error) {
	page := Page{Number: 1, Size: defaultPageSize}
	if number != nil {
		if *number < 1 {
			return Page{}, invalid("page", "invalid_page")
		}
		page.Number = *number
	}
	if size != nil {
		if *size < 1 || *size > maxPageSize {
			return Page{}, invalid("page_size", "invalid_page_size")
		}
		page.Size = *size
	}
	return page, nil
}

// Offset returns the zero-based row offset of the page.
func (p Page) Offset() int { return (p.Number - 1) * p.Size }

// DealFilter narrows a deal listing. Empty values mean "no restriction".
type DealFilter struct {
	Merchant    string
	City        string
	Category    string
	Subcategory string
	Query       string
	MinPrice    *int64
	MaxPrice    *int64
	MinRating   *float64
	MinDiscount *int
	EndingSoon  bool
}

// SearchQuery normalizes free text input and rejects over-long searches.
func SearchQuery(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if len([]rune(trimmed)) > maxSearchLength {
		return "", invalid("q", "search_too_long")
	}
	return trimmed, nil
}

// HomepageRow is a homepage section before resolution: a curated block with its raw targets.
type HomepageRow struct {
	ID        uuid.UUID
	Key       string
	Title     Text
	Type      string
	DealIDs   []uuid.UUID
	Rule      map[string]any
	SortOrder int
}

// SuggestionRow is a matched entity before the domain builds its label-free link.
type SuggestionRow struct {
	Type  string
	Slug  string
	Label string
}

// Repository is the persistence port. Implementations must return only publishable data where the
// method name says "Published" and must preserve no ordering promises beyond what is documented.
type Repository interface {
	ListCities(ctx context.Context) ([]City, error)
	ListCategories(ctx context.Context) ([]Category, error)

	SearchDealIDs(ctx context.Context, filter DealFilter, sort Sort, page Page) ([]uuid.UUID, error)
	CountDeals(ctx context.Context, filter DealFilter) (int64, error)
	DealSummaries(ctx context.Context, ids []uuid.UUID) ([]DealSummary, error)
	DealBySlug(ctx context.Context, slug string) (Deal, error)
	RelatedDealIDs(ctx context.Context, subcategoryID, excludeID uuid.UUID, citySlug string, limit int) ([]uuid.UUID, error)
	DealReviews(ctx context.Context, dealID uuid.UUID, page Page) ([]Review, int64, error)

	MerchantBySlug(ctx context.Context, slug string) (Merchant, error)
	MerchantReviews(ctx context.Context, merchantID uuid.UUID, limit int) ([]Review, error)

	SuggestDeals(ctx context.Context, query, locale string, limit int) ([]SuggestionRow, error)
	SuggestMerchants(ctx context.Context, query, locale string, limit int) ([]SuggestionRow, error)
	SuggestCategories(ctx context.Context, query, locale string, limit int) ([]SuggestionRow, error)

	HomepageRows(ctx context.Context) ([]HomepageRow, error)
}

// Cache is an optional read-through cache. Invalidate makes every cached entry stale at once.
type Cache interface {
	Get(ctx context.Context, key string, dst any) (bool, error)
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Invalidate(ctx context.Context) error
}

// Service implements the public catalog use cases.
type Service struct {
	repo  Repository
	cache Cache
	now   func() time.Time
}

// NewService builds the catalog service. A nil cache disables caching.
func NewService(repo Repository, cache Cache, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{repo: repo, cache: cache, now: now}
}

// Invalidate drops all cached catalog reads after content changes.
func (s *Service) Invalidate(ctx context.Context) error {
	if s.cache == nil {
		return nil
	}
	return s.cache.Invalidate(ctx)
}

// cached returns the value for key, loading and storing it on a miss. Cache failures never fail
// the request: the loader is the source of truth.
func cached[T any](ctx context.Context, s *Service, key string, ttl time.Duration, load func() (T, error)) (T, error) {
	var value T
	if s.cache != nil {
		if hit, err := s.cache.Get(ctx, key, &value); err == nil && hit {
			return value, nil
		}
	}
	value, err := load()
	if err != nil {
		return value, err
	}
	if s.cache != nil {
		_ = s.cache.Set(ctx, key, value, ttl)
	}
	return value, nil
}

// PublicConfig returns the bootstrap payload: locales, currencies, cities, categories and rate.
func (s *Service) PublicConfig(ctx context.Context) (PublicConfig, error) {
	return cached(ctx, s, "config", cacheTTLStatic, func() (PublicConfig, error) {
		cities, err := s.Cities(ctx)
		if err != nil {
			return PublicConfig{}, err
		}
		categories, err := s.Categories(ctx)
		if err != nil {
			return PublicConfig{}, err
		}
		return PublicConfig{
			Locales:    []string{"uz", "ru", "en"},
			Currencies: []string{"UZS"},
			Cities:     cities,
			Categories: categories,
		}, nil
	})
}

// Cities lists active cities.
func (s *Service) Cities(ctx context.Context) ([]City, error) {
	return cached(ctx, s, "cities", cacheTTLStatic, func() ([]City, error) {
		return s.repo.ListCities(ctx)
	})
}

// Categories lists active categories with their subcategories.
func (s *Service) Categories(ctx context.Context) ([]Category, error) {
	return cached(ctx, s, "categories", cacheTTLStatic, func() ([]Category, error) {
		return s.repo.ListCategories(ctx)
	})
}

// Homepage resolves active homepage sections into publishable content. Sections are global in
// Phase 3; the city only narrows the default location context for later per-city curation.
func (s *Service) Homepage(ctx context.Context) ([]HomepageSection, error) {
	return cached(ctx, s, "homepage", cacheTTL, func() ([]HomepageSection, error) {
		rows, err := s.repo.HomepageRows(ctx)
		if err != nil {
			return nil, err
		}
		sections := make([]HomepageSection, 0, len(rows))
		for _, row := range rows {
			section, ok, err := s.resolveSection(ctx, row)
			if err != nil {
				return nil, err
			}
			if ok {
				sections = append(sections, section)
			}
		}
		return sections, nil
	})
}

func (s *Service) resolveSection(ctx context.Context, row HomepageRow) (HomepageSection, bool, error) {
	section := HomepageSection{ID: row.ID, Key: row.Key, Title: row.Title, Type: row.Type, SortOrder: row.SortOrder}
	switch row.Type {
	case "deals":
		deals, err := s.publishedSummaries(ctx, row.DealIDs)
		if err != nil {
			return section, false, err
		}
		section.Deals = deals
		return section, len(deals) > 0, nil
	case "categories":
		categories, err := s.Categories(ctx)
		if err != nil {
			return section, false, err
		}
		section.Categories = categories
		return section, len(categories) > 0, nil
	default:
		// Banners have their own placement contract and are not rendered as deal sections.
		return section, false, nil
	}
}

// ListDeals returns one page of publishable deals for the filter and sort.
func (s *Service) ListDeals(ctx context.Context, filter DealFilter, sort Sort, page Page) (DealPage, error) {
	key := fmt.Sprintf("deals:%s:%s:%d:%d", sort, keyOf(filter), page.Number, page.Size)
	return cached(ctx, s, key, cacheTTL, func() (DealPage, error) {
		ids, err := s.repo.SearchDealIDs(ctx, filter, sort, page)
		if err != nil {
			return DealPage{}, fmt.Errorf("search deals: %w", err)
		}
		total, err := s.repo.CountDeals(ctx, filter)
		if err != nil {
			return DealPage{}, fmt.Errorf("count deals: %w", err)
		}
		items, err := s.publishedSummaries(ctx, ids)
		if err != nil {
			return DealPage{}, err
		}
		return DealPage{PageInfo: PageInfo{Page: page.Number, PageSize: page.Size, Total: total}, Items: items}, nil
	})
}

// Deal returns the deal detail. Expired, scheduled and sold-out deals stay visible for SEO and
// their own states; drafts, pending and rejected deals never reach this method.
func (s *Service) Deal(ctx context.Context, slug string) (Deal, error) {
	return cached(ctx, s, "deal:"+slug, cacheTTL, func() (Deal, error) {
		return s.repo.DealBySlug(ctx, slug)
	})
}

// RelatedDeals returns publishable deals in the same subcategory and city.
func (s *Service) RelatedDeals(ctx context.Context, slug, citySlug string) (DealPage, error) {
	key := fmt.Sprintf("related:%s:%s", slug, citySlug)
	return cached(ctx, s, key, cacheTTL, func() (DealPage, error) {
		deal, err := s.repo.DealBySlug(ctx, slug)
		if err != nil {
			return DealPage{}, err
		}
		ids, err := s.repo.RelatedDealIDs(ctx, deal.Subcategory.ID, deal.ID, citySlug, relatedLimit)
		if err != nil {
			return DealPage{}, fmt.Errorf("list related deals: %w", err)
		}
		items, err := s.publishedSummaries(ctx, ids)
		if err != nil {
			return DealPage{}, err
		}
		return DealPage{PageInfo: PageInfo{Page: 1, PageSize: relatedLimit, Total: int64(len(items))}, Items: items}, nil
	})
}

// DealReviews returns one page of published reviews for a deal.
func (s *Service) DealReviews(ctx context.Context, slug string, page Page) (ReviewPage, error) {
	key := fmt.Sprintf("reviews:%s:%d:%d", slug, page.Number, page.Size)
	return cached(ctx, s, key, cacheTTL, func() (ReviewPage, error) {
		deal, err := s.repo.DealBySlug(ctx, slug)
		if err != nil {
			return ReviewPage{}, err
		}
		items, total, err := s.repo.DealReviews(ctx, deal.ID, page)
		if err != nil {
			return ReviewPage{}, fmt.Errorf("list deal reviews: %w", err)
		}
		return ReviewPage{PageInfo: PageInfo{Page: page.Number, PageSize: page.Size, Total: total}, Items: items}, nil
	})
}

// ReviewPage is one page of reviews.
type ReviewPage struct {
	PageInfo
	Items []Review `json:"items"`
}

// Merchant returns an active merchant profile with its publishable deals and recent reviews.
func (s *Service) Merchant(ctx context.Context, slug string) (MerchantProfile, error) {
	return cached(ctx, s, "merchant:"+slug, cacheTTL, func() (MerchantProfile, error) {
		merchant, err := s.repo.MerchantBySlug(ctx, slug)
		if err != nil {
			return MerchantProfile{}, err
		}
		ids, err := s.repo.SearchDealIDs(ctx, DealFilter{Merchant: slug}, SortRecommended, Page{Number: 1, Size: maxPageSize})
		if err != nil {
			return MerchantProfile{}, fmt.Errorf("list merchant deals: %w", err)
		}
		deals, err := s.publishedSummaries(ctx, ids)
		if err != nil {
			return MerchantProfile{}, err
		}
		reviews, err := s.repo.MerchantReviews(ctx, merchant.ID, merchantReviews)
		if err != nil {
			return MerchantProfile{}, fmt.Errorf("list merchant reviews: %w", err)
		}
		return MerchantProfile{Merchant: merchant, Deals: deals, Reviews: reviews}, nil
	})
}

// Suggestions returns deal, merchant and category suggestions for the search bar. Results are
// not cached: they respond to every keystroke and the queries are indexed.
func (s *Service) Suggestions(ctx context.Context, rawQuery, locale string) ([]Suggestion, error) {
	query, err := SearchQuery(rawQuery)
	if err != nil {
		return nil, err
	}
	if len([]rune(query)) < 2 {
		return nil, invalid("q", "query_too_short")
	}
	deals, err := s.repo.SuggestDeals(ctx, query, locale, 5)
	if err != nil {
		return nil, fmt.Errorf("suggest deals: %w", err)
	}
	merchants, err := s.repo.SuggestMerchants(ctx, query, locale, 3)
	if err != nil {
		return nil, fmt.Errorf("suggest merchants: %w", err)
	}
	categories, err := s.repo.SuggestCategories(ctx, query, locale, 2)
	if err != nil {
		return nil, fmt.Errorf("suggest categories: %w", err)
	}
	items := make([]Suggestion, 0, maxSuggestions)
	for _, row := range append(append(deals, merchants...), categories...) {
		items = append(items, Suggestion{Type: row.Type, Label: row.Label, Href: suggestionHref(row)})
		if len(items) == maxSuggestions {
			break
		}
	}
	return items, nil
}

func suggestionHref(row SuggestionRow) string {
	switch row.Type {
	case "deal":
		return "/deal/" + row.Slug
	case "merchant":
		return "/business/" + row.Slug
	default:
		return "/" + row.Slug
	}
}

// keyOf is a stable cache-key fragment for a filter; JSON prints pointer values, not addresses.
func keyOf(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "unkeyed"
	}
	return string(encoded)
}

// publishedSummaries loads the given deals and keeps only those a visitor may see, in input order.
func (s *Service) publishedSummaries(ctx context.Context, ids []uuid.UUID) ([]DealSummary, error) {
	if len(ids) == 0 {
		return []DealSummary{}, nil
	}
	loaded, err := s.repo.DealSummaries(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("load deal summaries: %w", err)
	}
	byID := make(map[uuid.UUID]DealSummary, len(loaded))
	for _, deal := range loaded {
		byID[deal.ID] = deal
	}
	now := s.now()
	result := make([]DealSummary, 0, len(ids))
	for _, id := range ids {
		deal, ok := byID[id]
		if ok && IsPublishable(deal, now) {
			result = append(result, deal)
		}
	}
	return result, nil
}

// IsPublishable reports whether a summary may appear in listings.
func IsPublishable(deal DealSummary, now time.Time) bool {
	return deal.Status == "active" && deal.EndsAt.After(now)
}

// IsEndingSoon reports whether urgency copy is justified for a deal.
func IsEndingSoon(deal DealSummary, now time.Time) bool {
	return deal.EndsAt.After(now) && deal.EndsAt.Sub(now) <= endingSoonWindow
}
