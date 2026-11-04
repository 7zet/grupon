package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/grupon/grupon/services/api/internal/domain/catalog"
)

// stubRepository serves one published deal and one city; methods the tests do not reach are unused.
type stubRepository struct {
	catalog.Repository
	deal catalog.DealSummary
}

func (s *stubRepository) ListCities(context.Context) ([]catalog.City, error) {
	return []catalog.City{{
		ID:   uuid.New(),
		Slug: "tashkent",
		Name: catalog.Text{Uz: "Toshkent", Ru: "Ташкент", En: "Tashkent"},
		Lat:  41.2995,
		Lng:  69.2401,
	}}, nil
}

func (s *stubRepository) SearchDealIDs(context.Context, catalog.DealFilter, catalog.Sort, catalog.Page) ([]uuid.UUID, error) {
	return []uuid.UUID{s.deal.ID}, nil
}

func (s *stubRepository) CountDeals(context.Context, catalog.DealFilter) (int64, error) {
	return 1, nil
}

func (s *stubRepository) DealSummaries(context.Context, []uuid.UUID) ([]catalog.DealSummary, error) {
	return []catalog.DealSummary{s.deal}, nil
}

func (s *stubRepository) DealBySlug(_ context.Context, slug string) (catalog.Deal, error) {
	if slug != s.deal.Slug {
		return catalog.Deal{}, catalog.ErrNotFound
	}
	return catalog.Deal{DealSummary: s.deal}, nil
}

func (s *stubRepository) IncrementPromoCodeClicks(_ context.Context, id uuid.UUID) (bool, error) {
	return id == s.deal.ID, nil
}

// newTestRouter builds the production router around the stub repository. invalidations counts
// how often the internal invalidation hook actually ran.
func newTestRouter(t *testing.T, token string, invalidations *int) http.Handler {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	deal := catalog.DealSummary{
		ID:       uuid.New(),
		Slug:     "spa-day",
		Status:   "active",
		Title:    catalog.Text{Uz: "Spa kuni", Ru: "СПА", En: "Spa day"},
		Merchant: catalog.Merchant{ID: uuid.New(), Slug: "lotus", DisplayName: "Lotus"},
		Category: catalog.Category{ID: uuid.New(), Slug: "beauty-and-spas", Icon: "spa"},
		Options: []catalog.Option{{
			ID:            uuid.New(),
			Title:         catalog.Text{En: "Basic"},
			OriginalPrice: catalog.Money{Amount: 250000},
			DealPrice:     catalog.Money{Amount: 150000},
			StockTotal:    10,
			IsActive:      true,
		}},
		EndsAt: time.Now().Add(48 * time.Hour),
	}
	service := catalog.NewService(&stubRepository{deal: deal}, nil, time.Now)
	return NewRouter(RouterOptions{
		Logger:  logger,
		Catalog: NewCatalogAPI(service, logger, false),
		Internal: NewInternalAPI(token, func(context.Context) error {
			*invalidations++
			return nil
		}),
	})
}

func send(t *testing.T, handler http.Handler, method, target string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, target, nil)
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func problemCode(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()
	var problem struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &problem); err != nil {
		t.Fatalf("decode problem: %v; body=%s", err, recorder.Body.String())
	}
	return problem.Code
}

func TestCatalogEndpointsMapDomainResultsAndProblems(t *testing.T) {
	t.Parallel()
	invalidations := 0
	router := newTestRouter(t, "secret-token", &invalidations)

	t.Run("cities are served as a contract list", func(t *testing.T) {
		recorder := send(t, router, http.MethodGet, "/api/v1/cities", nil)
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d", recorder.Code)
		}
		if !strings.Contains(recorder.Body.String(), `"slug":"tashkent"`) {
			t.Errorf("body = %s", recorder.Body.String())
		}
	})

	t.Run("deal prices are integer minor units with a currency", func(t *testing.T) {
		recorder := send(t, router, http.MethodGet, "/api/v1/deals?city=tashkent&sort=price_asc", nil)
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"deal_price":{"amount":150000,"currency":"UZS"}`) {
			t.Errorf("money shape wrong: %s", recorder.Body.String())
		}
	})

	t.Run("invalid sort is a 400 problem with a stable code", func(t *testing.T) {
		recorder := send(t, router, http.MethodGet, "/api/v1/deals?sort=bogus", nil)
		if recorder.Code != http.StatusBadRequest || problemCode(t, recorder) != "invalid_sort" {
			t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
		}
		if got := recorder.Header().Get("Content-Type"); got != "application/problem+json" {
			t.Errorf("content type = %s", got)
		}
	})

	t.Run("malformed numbers are rejected by binding with a problem", func(t *testing.T) {
		recorder := send(t, router, http.MethodGet, "/api/v1/deals?page=abc", nil)
		if recorder.Code != http.StatusBadRequest || problemCode(t, recorder) != "invalid_parameter" {
			t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
		}
	})

	t.Run("missing deal is a 404 with the deal-specific code", func(t *testing.T) {
		recorder := send(t, router, http.MethodGet, "/api/v1/deals/missing", nil)
		if recorder.Code != http.StatusNotFound || problemCode(t, recorder) != "deal_not_found" {
			t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
		}
	})

	t.Run("operations outside the catalog stay unimplemented", func(t *testing.T) {
		recorder := send(t, router, http.MethodGet, "/api/v1/me", nil)
		if recorder.Code != http.StatusNotImplemented {
			t.Fatalf("status = %d, want 501", recorder.Code)
		}
	})

	t.Run("internal invalidation requires the shared token", func(t *testing.T) {
		recorder := send(t, router, http.MethodPost, "/internal/cache/invalidate", nil)
		if recorder.Code != http.StatusUnauthorized || invalidations != 0 {
			t.Fatalf("unauthenticated status=%d invalidations=%d", recorder.Code, invalidations)
		}
		recorder = send(t, router, http.MethodPost, "/internal/cache/invalidate",
			map[string]string{internalTokenHeader: "secret-token"})
		if recorder.Code != http.StatusAccepted || invalidations != 1 {
			t.Fatalf("authenticated status=%d invalidations=%d", recorder.Code, invalidations)
		}
	})
}
