package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/grupon/grupon/services/api/internal/domain/catalog"
	"github.com/grupon/grupon/services/api/internal/domain/merchant"
	"github.com/grupon/grupon/services/api/internal/domain/moderation"
	"github.com/grupon/grupon/services/api/internal/store"
)

// auditCount counts audit rows for one action on one entity.
func auditCount(t *testing.T, fixture catalogFixture, action string, entityID uuid.UUID) int {
	t.Helper()
	var count int
	if err := fixture.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM audit_logs WHERE action = $1 AND entity_id = $2`, action, entityID).Scan(&count); err != nil {
		t.Fatalf("count audit rows: %v", err)
	}
	return count
}

func seededID(t *testing.T, fixture catalogFixture, query string, args ...any) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := fixture.pool.QueryRow(context.Background(), query, args...).Scan(&id); err != nil {
		t.Fatalf("seed lookup: %v", err)
	}
	return id
}

// pendingDeal inserts a complete deal under review for the seeded owner, the way a submitted draft looks.
func pendingDeal(t *testing.T, fixture catalogFixture, ownerID, merchantID uuid.UUID) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	categoryID := seededID(t, fixture, `SELECT id FROM categories WHERE searchable ORDER BY sort_order LIMIT 1`)
	subcategoryID := seededID(t, fixture, `SELECT id FROM subcategories WHERE category_id = $1 ORDER BY sort_order LIMIT 1`, categoryID)
	locationID := seededID(t, fixture, `SELECT id FROM merchant_locations WHERE merchant_id = $1 LIMIT 1`, merchantID)
	text := catalog.Text{Uz: "matn", Ru: "текст", En: "text"}
	days := 30
	in := merchant.DealInput{
		Slug: "audit-" + uuid.NewString(), CategoryID: categoryID, SubcategoryID: subcategoryID,
		Title: text, ShortDescription: text, Description: text, Highlights: []catalog.Text{text, text, text},
		Terms: text, RedemptionMethod: "qr_or_code", RedemptionInstructions: text,
		StartsAt: fixture.now.Add(-time.Hour), EndsAt: fixture.now.Add(30 * 24 * time.Hour), VoucherValidDays: &days,
		MaxPerUser: 1, LocationIDs: []uuid.UUID{locationID},
		Options: []merchant.DealOptionData{{Title: text, OriginalPrice: 200000, DealPrice: 100000, StockTotal: 5, IsActive: true}},
		Media: []merchant.DealMediaData{
			{URL: "/media/1.jpg", Alt: text, SortOrder: 0, Width: 1, Height: 1}, {URL: "/media/2.jpg", Alt: text, SortOrder: 1, Width: 1, Height: 1},
			{URL: "/media/3.jpg", Alt: text, SortOrder: 2, Width: 1, Height: 1},
		},
	}
	dealID, err := store.NewMerchantDealRepository(fixture.pool, store.NewCatalogRepository(fixture.pool)).Insert(ctx, merchantID,
		merchant.DealWrite{Input: in, Status: merchant.DealInReview, Fingerprint: merchant.ReviewFingerprint(in)},
		merchant.AuditEntry{ActorID: ownerID, Role: merchant.RoleOwner, Action: "deal_submitted", After: map[string]any{}})
	if err != nil {
		t.Fatalf("insert pending deal: %v", err)
	}
	return dealID
}

func TestEveryStaffWriteLeavesAnAuditRow(t *testing.T) {
	fixture := newCatalogFixture(t)
	ctx := context.Background()
	ownerID := seededID(t, fixture, `SELECT id FROM users WHERE email = 'owner@local.test'`)
	adminID := seededID(t, fixture, `SELECT id FROM users WHERE email = 'admin@local.test'`)
	merchantID := seededID(t, fixture, `SELECT merchant_id FROM role_assignments WHERE user_id = $1 AND merchant_id IS NOT NULL LIMIT 1`, ownerID)
	dealRepo := store.NewMerchantDealRepository(fixture.pool, store.NewCatalogRepository(fixture.pool))
	moderationService := moderation.NewService(
		store.NewModerationRepository(fixture.pool, store.NewCatalogRepository(fixture.pool), dealRepo), time.Now)
	admin := moderation.Actor{UserID: adminID, Role: "superadmin"}

	approved := pendingDeal(t, fixture, ownerID, merchantID)
	if _, err := moderationService.Approve(ctx, admin, approved); err != nil {
		t.Fatalf("approve: %v", err)
	}
	if n := auditCount(t, fixture, "deal_approved", approved); n != 1 {
		t.Fatalf("deal_approved audit rows = %d, want 1", n)
	}

	rejected := pendingDeal(t, fixture, ownerID, merchantID)
	if _, err := moderationService.Reject(ctx, admin, rejected, "Photos do not match the offer"); err != nil {
		t.Fatalf("reject: %v", err)
	}
	if n := auditCount(t, fixture, "deal_rejected", rejected); n != 1 {
		t.Fatalf("deal_rejected audit rows = %d, want 1", n)
	}

	reviewID := seededID(t, fixture, `SELECT id FROM reviews WHERE deleted_at IS NULL AND user_id <> $1 LIMIT 1`, ownerID)
	reportIDs := make([]uuid.UUID, 0, 2)
	for _, reporter := range []uuid.UUID{ownerID, adminID} {
		var reportID uuid.UUID
		if err := fixture.pool.QueryRow(ctx,
			`INSERT INTO review_reports (review_id, reporter_user_id, reason) VALUES ($1, $2, 'Spam, not a real visit') RETURNING id`,
			reviewID, reporter).Scan(&reportID); err != nil {
			t.Fatalf("insert report: %v", err)
		}
		reportIDs = append(reportIDs, reportID)
	}
	reports := moderation.NewReportService(store.NewAdminReportRepository(fixture.pool, store.NewReviewRepository(fixture.pool)))
	if err := reports.Dismiss(ctx, admin, reportIDs[0]); err != nil {
		t.Fatalf("dismiss: %v", err)
	}
	if n := auditCount(t, fixture, "review_report_dismissed", reportIDs[0]); n != 1 {
		t.Fatalf("dismiss audit rows = %d, want 1", n)
	}
	if err := reports.Hide(ctx, admin, reviewID); err != nil {
		t.Fatalf("hide: %v", err)
	}
	if n := auditCount(t, fixture, "review_hidden", reviewID); n != 1 {
		t.Fatalf("hide audit rows = %d, want 1", n)
	}
}

func TestModeratorCannotReadFinanceOrAuditWrites(t *testing.T) {
	fixture := newCatalogFixture(t)
	ctx := context.Background()
	ownerID := seededID(t, fixture, `SELECT id FROM users WHERE email = 'owner@local.test'`)
	merchantID := seededID(t, fixture, `SELECT merchant_id FROM role_assignments WHERE user_id = $1 AND merchant_id IS NOT NULL LIMIT 1`, ownerID)
	dealRepo := store.NewMerchantDealRepository(fixture.pool, store.NewCatalogRepository(fixture.pool))
	service := moderation.NewService(store.NewModerationRepository(fixture.pool, store.NewCatalogRepository(fixture.pool), dealRepo), time.Now)
	deal := pendingDeal(t, fixture, ownerID, merchantID)

	// Finance and support hold no moderation rights.
	for _, role := range []string{"finance", "support"} {
		_, err := service.Approve(ctx, moderation.Actor{UserID: uuid.New(), Role: role}, deal)
		if !errors.Is(err, moderation.ErrForbidden) {
			t.Fatalf("%s approve: got %v", role, err)
		}
	}
	if n := auditCount(t, fixture, "deal_approved", deal); n != 0 {
		t.Fatalf("refused approve still wrote %d audit rows", n)
	}

	// Moderators can moderate but cannot read the audit trail that records their own decisions.
	reports := moderation.NewReportService(store.NewAdminReportRepository(fixture.pool, store.NewReviewRepository(fixture.pool)))
	if _, _, err := reports.Audit(ctx, moderation.Actor{UserID: uuid.New(), Role: "moderator"}, catalog.Page{Number: 1, Size: 20}); !errors.Is(err, moderation.ErrAuditForbidden) {
		t.Fatalf("moderator audit read: got %v", err)
	}
}
