package merchant

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/grupon/grupon/services/api/internal/domain/catalog"
)

type fakeDealRepo struct {
	record  DealRecord
	writes  []DealWrite
	status  string
	deleted bool
}

func (f *fakeDealRepo) References(context.Context, uuid.UUID, DealInput) error { return nil }

func (f *fakeDealRepo) Insert(_ context.Context, _ uuid.UUID, write DealWrite, _ AuditEntry) (uuid.UUID, error) {
	f.writes = append(f.writes, write)
	return uuid.New(), nil
}

func (f *fakeDealRepo) Update(_ context.Context, _, _ uuid.UUID, write DealWrite, _ AuditEntry) error {
	f.writes = append(f.writes, write)
	return nil
}

func (f *fakeDealRepo) Record(context.Context, uuid.UUID, uuid.UUID) (DealRecord, error) {
	return f.record, nil
}

func (f *fakeDealRepo) SetStatus(_ context.Context, _, _ uuid.UUID, status string, _ AuditEntry) error {
	f.status = status
	return nil
}

func (f *fakeDealRepo) SoftDeleteDraft(context.Context, uuid.UUID, uuid.UUID, AuditEntry) (bool, error) {
	f.deleted = true
	return true, nil
}

func (f *fakeDealRepo) Detail(context.Context, uuid.UUID, uuid.UUID) (catalog.Deal, error) {
	return catalog.Deal{}, nil
}

func (f *fakeDealRepo) ListIDs(context.Context, uuid.UUID, string, catalog.Page) ([]uuid.UUID, int64, error) {
	return nil, 0, nil
}

func (f *fakeDealRepo) Summaries(context.Context, []uuid.UUID) ([]catalog.DealSummary, error) {
	return nil, nil
}

func validDeal() DealInput {
	days := 30
	text := catalog.Text{Uz: "a", Ru: "б", En: "a"}
	return DealInput{
		Slug:                   "spa-day",
		CategoryID:             uuid.New(),
		SubcategoryID:          uuid.New(),
		Kind:                   "voucher",
		Title:                  text,
		ShortDescription:       text,
		Description:            text,
		Highlights:             []catalog.Text{text, text, text},
		Terms:                  text,
		RedemptionMethod:       "qr_or_code",
		RedemptionInstructions: text,
		StartsAt:               time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndsAt:                 time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		VoucherValidDays:       &days,
		MaxPerUser:             2,
		LocationIDs:            []uuid.UUID{uuid.New()},
		Options: []DealOptionData{{
			Title: text, OriginalPrice: 200000, DealPrice: 100000, StockTotal: 10, IsActive: true,
		}},
		Media: []DealMediaData{
			{URL: "/media/a.jpg", Alt: text, Width: 800, Height: 600}, {URL: "/media/b.jpg", Alt: text, Width: 800, Height: 600},
			{URL: "/media/c.jpg", Alt: text, Width: 800, Height: 600},
		},
	}
}

func TestDealValidationRules(t *testing.T) {
	cases := map[string]func(*DealInput){
		"highlights":   func(in *DealInput) { in.Highlights = in.Highlights[:2] },
		"media":        func(in *DealInput) { in.Media = in.Media[:2] },
		"price":        func(in *DealInput) { in.Options[0].DealPrice = in.Options[0].OriginalPrice + 1 },
		"validity":     func(in *DealInput) { in.VoucherValidDays = nil },
		"order":        func(in *DealInput) { in.EndsAt = in.StartsAt },
		"locations":    func(in *DealInput) { in.LocationIDs = nil },
		"slug":         func(in *DealInput) { in.Slug = "Spa Day" },
		"localization": func(in *DealInput) { in.Terms.Ru = "" },
	}
	for name, mutate := range cases {
		in := validDeal()
		mutate(&in)
		if err := in.Validate(); err == nil {
			t.Fatalf("%s: expected validation error", name)
		}
	}
	if err := validDeal().Validate(); err != nil {
		t.Fatalf("valid deal rejected: %v", err)
	}
}

func TestStockOnlyEditKeepsReviewFingerprint(t *testing.T) {
	base := validDeal()
	stockOnly := validDeal()
	stockOnly.Options[0].StockTotal = 500
	if ReviewFingerprint(base) != ReviewFingerprint(stockOnly) {
		t.Fatal("stock change must not change the review fingerprint")
	}
	priceChange := validDeal()
	priceChange.Options[0].DealPrice = 90000
	if ReviewFingerprint(base) == ReviewFingerprint(priceChange) {
		t.Fatal("price change must change the review fingerprint")
	}
}

func TestLiveDealPriceEditReturnsToReview(t *testing.T) {
	repo := &fakeDealRepo{record: DealRecord{ID: uuid.New(), Status: DealActive, Fingerprint: ReviewFingerprint(validDeal())}}
	service := NewDealService(repo, time.Now)
	actor := Actor{UserID: uuid.New(), Role: RoleOwner}

	stock := validDeal()
	stock.Options[0].StockTotal = 3
	if _, err := service.Update(context.Background(), uuid.New(), repo.record.ID, actor, stock); err != nil {
		t.Fatal(err)
	}
	if got := repo.writes[0].Status; got != DealActive {
		t.Fatalf("stock-only edit changed status to %s", got)
	}

	price := validDeal()
	price.Options[0].DealPrice = 90000
	if _, err := service.Update(context.Background(), uuid.New(), repo.record.ID, actor, price); err != nil {
		t.Fatal(err)
	}
	if got := repo.writes[1].Status; got != DealPendingReview {
		t.Fatalf("price edit status = %s, want pending_review", got)
	}
}

func TestSubmitAndDeleteRespectWorkflowState(t *testing.T) {
	repo := &fakeDealRepo{record: DealRecord{ID: uuid.New(), Status: DealActive}}
	service := NewDealService(repo, time.Now)
	actor := Actor{UserID: uuid.New(), Role: RoleManager}

	if _, err := service.Submit(context.Background(), uuid.New(), repo.record.ID, actor); !errors.Is(err, ErrDealStateInvalid) {
		t.Fatalf("submit active deal: got %v", err)
	}
	if err := service.Delete(context.Background(), uuid.New(), repo.record.ID, actor); err != nil {
		t.Fatal(err)
	}

	repo.record.Status = DealDraft
	if _, err := service.Submit(context.Background(), uuid.New(), repo.record.ID, actor); err != nil {
		t.Fatal(err)
	}
	if repo.status != DealPendingReview {
		t.Fatalf("submitted status = %s", repo.status)
	}
}
