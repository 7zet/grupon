package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/grupon/grupon/services/api/internal/domain/finance"
	"github.com/grupon/grupon/services/api/internal/domain/merchant"
	"github.com/grupon/grupon/services/api/internal/domain/redemption"
	"github.com/grupon/grupon/services/api/internal/store"
)

// AT-09: a cashier asks for a payout. The request is refused and the refusal is written to the audit log.
func TestAT09CashierCannotPrepareAPayoutAndTheAttemptIsAudited(t *testing.T) {
	env := newATEnv(t, 1)
	merchantID, _ := env.merchantAndLocation(t)
	cashierID := seededID(t, env.fixture, `SELECT id FROM users WHERE email = 'cashier@local.test'`)

	service := finance.NewService(store.NewFinanceRepository(env.fixture.pool), time.Now)
	_, err := service.PreparePayout(env.ctx, finance.Actor{UserID: cashierID, Role: "merchant_cashier"}, merchantID)
	if !errors.Is(err, finance.ErrForbidden) {
		t.Fatalf("cashier payout: got %v, want ErrForbidden", err)
	}
	if got := env.count(t, `SELECT count(*) FROM audit_logs WHERE action = 'payout_prepare_denied' AND actor_user_id = $1`, cashierID); got != 1 {
		t.Fatalf("audit rows for the refused payout = %d, want 1", got)
	}
}

// AT-10: a closed category publishes nothing. The merchant gets category_closed; an open category passes this gate.
func TestAT10ClosedCategoryRefusesADealWithAReason(t *testing.T) {
	env := newATEnv(t, 1)
	merchantID, _ := env.merchantAndLocation(t)
	repo := store.NewMerchantDealRepository(env.fixture.pool, store.NewCatalogRepository(env.fixture.pool))

	closed := categoryBySlug(t, env, "automotive")
	err := repo.References(env.ctx, merchantID, merchant.DealInput{CategoryID: closed, SubcategoryID: subcategoryOf(t, env, closed)})
	var invalid *merchant.InvalidInputError
	if !errors.As(err, &invalid) || invalid.Code != "category_closed" || invalid.Field != "category_id" {
		t.Fatalf("closed category: got %v, want category_closed on category_id", err)
	}

	open := categoryBySlug(t, env, "beauty-and-spas")
	if err := repo.References(env.ctx, merchantID, merchant.DealInput{CategoryID: open, SubcategoryID: subcategoryOf(t, env, open)}); err != nil {
		t.Fatalf("open category: got %v, want no error", err)
	}
}

// AT-16: a buyer opens a dispute ("this was not me"). The disputed redemption leaves the payout; the other stays.
func TestAT16OpenDisputeKeepsTheDisputedAmountOutOfThePayout(t *testing.T) {
	env := newATEnv(t, 2)
	merchantID, locationID := env.merchantAndLocation(t)
	redemptions := store.NewRedemptionRepository(env.fixture.pool, store.NewCommerceRepository(env.fixture.pool), store.NewCatalogRepository(env.fixture.pool))
	finances := store.NewFinanceRepository(env.fixture.pool)

	// Two redemptions made five days ago, so both are past the payout dispute window.
	var disputedVoucher, otherVoucher uuid.UUID
	for _, voucherID := range []*uuid.UUID{&disputedVoucher, &otherVoucher} {
		*voucherID = env.sellOneVoucher(t, env.newBuyer(t))
		audit := redemption.AuditEntry{ActorID: env.buyer, Role: "merchant_cashier", VoucherID: *voucherID, Action: "voucher_redeemed"}
		if ok, err := redemptions.UseVoucher(env.ctx, merchantID, *voucherID, locationID, time.Now(), audit, nil); err != nil || !ok {
			t.Fatalf("redeem: ok=%v err=%v", ok, err)
		}
		if _, err := env.fixture.pool.Exec(env.ctx, `
			UPDATE redemptions SET created_at = now() - interval '5 days' WHERE voucher_id = $1`, *voucherID); err != nil {
			t.Fatal(err)
		}
	}
	disputedRedemption := redemptionIDOf(t, env, disputedVoucher)
	otherRedemption := redemptionIDOf(t, env, otherVoucher)

	cutoff := time.Now()
	before := payableIDs(t, env.ctx, finances, merchantID, cutoff)
	if !before[disputedRedemption] || !before[otherRedemption] {
		t.Fatalf("both redemptions should be payable before the dispute: %v", before)
	}

	if ok, err := finances.OpenDispute(env.ctx, disputedVoucher, env.buyer, "not me"); err != nil || !ok {
		t.Fatalf("open dispute: ok=%v err=%v", ok, err)
	}

	after := payableIDs(t, env.ctx, finances, merchantID, cutoff)
	if after[disputedRedemption] {
		t.Fatal("disputed redemption is still payable")
	}
	if !after[otherRedemption] {
		t.Fatal("undisputed redemption left the payout")
	}
}

func categoryBySlug(t *testing.T, env atEnv, slug string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := env.fixture.pool.QueryRow(env.ctx, `SELECT id FROM categories WHERE slug = $1`, slug).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func subcategoryOf(t *testing.T, env atEnv, category uuid.UUID) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := env.fixture.pool.QueryRow(env.ctx, `SELECT id FROM subcategories WHERE category_id = $1 ORDER BY slug LIMIT 1`, category).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func redemptionIDOf(t *testing.T, env atEnv, voucherID uuid.UUID) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := env.fixture.pool.QueryRow(env.ctx, `SELECT id FROM redemptions WHERE voucher_id = $1`, voucherID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// payableIDs returns the redemption ids that are payable for the merchant before the cutoff.
func payableIDs(t *testing.T, ctx context.Context, repo *store.FinanceRepository, merchantID uuid.UUID, before time.Time) map[uuid.UUID]bool {
	t.Helper()
	items, err := repo.PayableRedemptions(ctx, merchantID, before)
	if err != nil {
		t.Fatal(err)
	}
	out := make(map[uuid.UUID]bool, len(items))
	for _, item := range items {
		out[item.ID] = true
	}
	return out
}
