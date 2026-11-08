package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/grupon/grupon/services/api/internal/domain/catalog"
	"github.com/grupon/grupon/services/api/internal/domain/commerce"
	"github.com/grupon/grupon/services/api/internal/domain/finance"
	"github.com/grupon/grupon/services/api/internal/domain/merchant"
	"github.com/grupon/grupon/services/api/internal/domain/moderation"
	"github.com/grupon/grupon/services/api/internal/domain/redemption"
	"github.com/grupon/grupon/services/api/internal/payments"
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

	ownerID := seededID(t, env.fixture, `SELECT id FROM users WHERE email='owner@local.test'`)
	dealID := pendingDeal(t, env.fixture, ownerID, merchantID)
	if _, err := env.fixture.pool.Exec(env.ctx, `UPDATE deals SET status='draft' WHERE id=$1`, dealID); err != nil {
		t.Fatal(err)
	}
	if _, err := env.fixture.pool.Exec(env.ctx, `UPDATE categories SET deal_creation_enabled=false WHERE id=$1`, open); err != nil {
		t.Fatal(err)
	}
	service := merchant.NewDealService(repo, time.Now)
	_, err = service.Submit(env.ctx, merchantID, dealID, merchant.Actor{UserID: ownerID, Role: merchant.RoleOwner})
	if !errors.As(err, &invalid) || invalid.Code != "category_closed" {
		t.Fatalf("submit after category closure: %v", err)
	}
}

func TestCategorySearchableHidesCategoryAndDeals(t *testing.T) {
	env := newATEnv(t, 1)
	var categoryID, dealID uuid.UUID
	if err := env.fixture.pool.QueryRow(env.ctx, `SELECT d.category_id,d.id FROM deals d JOIN deal_options o ON o.deal_id=d.id WHERE o.id=$1`, env.optionID).Scan(&categoryID, &dealID); err != nil {
		t.Fatal(err)
	}
	if _, err := env.fixture.pool.Exec(env.ctx, `UPDATE categories SET searchable=false WHERE id=$1`, categoryID); err != nil {
		t.Fatal(err)
	}
	repo := store.NewCatalogRepository(env.fixture.pool)
	categories, err := repo.ListCategories(env.ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range categories {
		if item.ID == categoryID {
			t.Fatal("non-searchable category was listed")
		}
	}
	ids, err := repo.SearchDealIDs(env.ctx, catalog.DealFilter{}, catalog.SortPopular, catalog.Page{Number: 1, Size: 100})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range ids {
		if id == dealID {
			t.Fatal("deal in non-searchable category was listed")
		}
	}
}

func TestCategoryPublishEnabledBlocksModerationApproval(t *testing.T) {
	fixture := newCatalogFixture(t)
	ctx := context.Background()
	ownerID := seededID(t, fixture, `SELECT id FROM users WHERE email='owner@local.test'`)
	adminID := seededID(t, fixture, `SELECT id FROM users WHERE email='admin@local.test'`)
	merchantID := seededID(t, fixture, `SELECT merchant_id FROM role_assignments WHERE user_id=$1 AND merchant_id IS NOT NULL LIMIT 1`, ownerID)
	dealID := pendingDeal(t, fixture, ownerID, merchantID)
	if _, err := fixture.pool.Exec(ctx, `UPDATE categories SET publish_enabled=false WHERE id=(SELECT category_id FROM deals WHERE id=$1)`, dealID); err != nil {
		t.Fatal(err)
	}
	deals := store.NewMerchantDealRepository(fixture.pool, store.NewCatalogRepository(fixture.pool))
	service := moderation.NewService(store.NewModerationRepository(fixture.pool, store.NewCatalogRepository(fixture.pool), deals), time.Now)
	if _, err := service.Approve(ctx, moderation.Actor{UserID: adminID, Role: "superadmin"}, dealID); !errors.Is(err, moderation.ErrCategoryPublishDisabled) {
		t.Fatalf("approve: %v", err)
	}
}

func TestCategoryPurchasableBlocksCheckout(t *testing.T) {
	env := newATEnv(t, 1)
	if _, err := env.fixture.pool.Exec(env.ctx, `UPDATE categories SET purchasable=false WHERE id=(SELECT d.category_id FROM deals d JOIN deal_options o ON o.deal_id=d.id WHERE o.id=$1)`, env.optionID); err != nil {
		t.Fatal(err)
	}
	_, _, err := env.service.CreateOrder(env.ctx, env.buyer, commerce.OrderInput{IdempotencyKey: uuid.NewString(), Items: []commerce.OrderItemInput{{OptionID: env.optionID, Quantity: 1}}})
	if !errors.Is(err, commerce.ErrCategoryNotPurchasable) {
		t.Fatalf("checkout: %v", err)
	}
}

// A buyer checks out one selected variant at a time. This also rejects a
// crafted cross-merchant request before any stock reservation is made.
func TestOrderRequiresExactlyOneVariant(t *testing.T) {
	env := newATEnv(t, 5)
	var other uuid.UUID
	if err := env.fixture.pool.QueryRow(env.ctx, `SELECT id FROM deal_options WHERE id <> $1 LIMIT 1`, env.optionID).Scan(&other); err != nil {
		t.Fatal(err)
	}
	for _, items := range [][]commerce.OrderItemInput{
		{{OptionID: env.optionID, Quantity: 1}, {OptionID: env.optionID, Quantity: 1}},
		{{OptionID: env.optionID, Quantity: 1}, {OptionID: other, Quantity: 1}},
	} {
		_, _, err := env.service.CreateOrder(env.ctx, env.buyer, commerce.OrderInput{IdempotencyKey: uuid.NewString(), Items: items})
		var validation *commerce.ValidationError
		if !errors.As(err, &validation) || validation.Field != "items" || validation.Code != "exactly_one_variant_required" {
			t.Fatalf("items=%d error = %v, want stable one-variant validation", len(items), err)
		}
	}
}

func TestCategoryRedeemableBlocksOnlyWhenFalse(t *testing.T) {
	env := newATEnv(t, 1)
	voucherID := env.sellOneVoucher(t, env.buyer)
	merchantID, locationID := env.merchantAndLocation(t)
	if _, err := env.fixture.pool.Exec(env.ctx, `UPDATE categories SET redeemable=false WHERE id=(SELECT category_id FROM deals WHERE id=(SELECT deal_id FROM vouchers WHERE id=$1))`, voucherID); err != nil {
		t.Fatal(err)
	}
	repo := store.NewRedemptionRepository(env.fixture.pool, store.NewCommerceRepository(env.fixture.pool), store.NewCatalogRepository(env.fixture.pool))
	_, err := repo.UseVoucher(env.ctx, merchantID, voucherID, locationID, time.Now(), redemption.AuditEntry{ActorID: env.buyer, Role: "merchant_cashier", VoucherID: voucherID, Action: "voucher_redeemed"}, nil)
	if !errors.Is(err, redemption.ErrCategoryNotRedeemable) {
		t.Fatalf("redeem: %v", err)
	}
}

func TestMerchantRedemptionHistoryScopesCashierAndMasksBuyer(t *testing.T) {
	env := newATEnv(t, 2)
	if _, err := env.fixture.pool.Exec(env.ctx, `UPDATE users SET name='Alisher Usmonov' WHERE id=$1`, env.buyer); err != nil {
		t.Fatal(err)
	}
	merchantID, locationID := env.merchantAndLocation(t)
	var before int64
	if err := env.fixture.pool.QueryRow(env.ctx, `SELECT count(*) FROM redemptions WHERE merchant_id=$1`, merchantID).Scan(&before); err != nil {
		t.Fatal(err)
	}
	cashierID := seededID(t, env.fixture, `SELECT id FROM users WHERE email='cashier@local.test'`)
	var cashierBefore int64
	if err := env.fixture.pool.QueryRow(env.ctx, `SELECT count(*) FROM redemptions WHERE merchant_id=$1 AND staff_user_id=$2`, merchantID, cashierID).Scan(&cashierBefore); err != nil {
		t.Fatal(err)
	}
	ownerID := seededID(t, env.fixture, `SELECT id FROM users WHERE email='owner@local.test'`)
	repo := store.NewRedemptionRepository(env.fixture.pool, store.NewCommerceRepository(env.fixture.pool), store.NewCatalogRepository(env.fixture.pool))
	for _, actorID := range []uuid.UUID{cashierID, ownerID} {
		voucherID := env.sellOneVoucher(t, env.buyer)
		ok, err := repo.UseVoucher(env.ctx, merchantID, voucherID, locationID, time.Now(), redemption.AuditEntry{ActorID: actorID, Role: "merchant_cashier", VoucherID: voucherID, Action: "voucher_redeemed"}, nil)
		if err != nil || !ok {
			t.Fatalf("redeem: ok=%v err=%v", ok, err)
		}
	}
	service := redemption.NewService(repo, nil, time.Now)
	page := catalog.Page{Number: 1, Size: 20}
	cashier, err := service.History(env.ctx, merchantID, merchant.Actor{UserID: cashierID, Role: merchant.RoleCashier}, page)
	if err != nil || cashier.Total != cashierBefore+1 || cashier.Items[0].Buyer != "Alisher U." {
		t.Fatalf("cashier history: %+v err=%v", cashier, err)
	}
	owner, err := service.History(env.ctx, merchantID, merchant.Actor{UserID: ownerID, Role: merchant.RoleOwner}, page)
	if err != nil || owner.Total != before+2 {
		t.Fatalf("owner history total=%d err=%v", owner.Total, err)
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

// AT-06: the buyer orders at the price of the current version. The merchant then changes the price. The order and
// its voucher keep the price the buyer agreed to.
func TestAT06ChangedPriceDoesNotChangeAPlacedOrder(t *testing.T) {
	env := newATEnv(t, 1)
	var agreed int64
	var agreedTerms []byte
	if err := env.fixture.pool.QueryRow(env.ctx, `SELECT deal_price FROM deal_options WHERE id = $1`, env.optionID).Scan(&agreed); err != nil {
		t.Fatal(err)
	}
	if err := env.fixture.pool.QueryRow(env.ctx, `SELECT d.terms FROM deals d JOIN deal_options o ON o.deal_id = d.id WHERE o.id = $1`, env.optionID).Scan(&agreedTerms); err != nil {
		t.Fatal(err)
	}
	order := env.order(t, 1)

	if _, err := env.fixture.pool.Exec(env.ctx, `UPDATE deal_options SET deal_price = deal_price + 5000 WHERE id = $1`, env.optionID); err != nil {
		t.Fatal(err)
	}
	if _, err := env.fixture.pool.Exec(env.ctx, `UPDATE deals SET terms = '{"uz":"Yangi shart","ru":"Новое условие","en":"New term"}' WHERE id = (SELECT deal_id FROM deal_options WHERE id = $1)`, env.optionID); err != nil {
		t.Fatal(err)
	}
	session, err := env.service.PayOrder(env.ctx, env.buyer, order.ID, env.mock.Name())
	if err != nil {
		t.Fatalf("pay order: %v", err)
	}
	if err := env.service.CompleteMockPayment(env.ctx, session.PaymentID, payments.OutcomeSucceeded); err != nil {
		t.Fatalf("complete payment: %v", err)
	}

	if got := env.count(t, `SELECT total::int FROM orders WHERE id = $1`, order.ID); int64(got) != agreed {
		t.Fatalf("order total = %d, want the agreed %d", got, agreed)
	}
	if got := env.count(t, `SELECT unit_price_snapshot::int FROM order_items WHERE order_id = $1`, order.ID); int64(got) != agreed {
		t.Fatalf("item price = %d, want the agreed %d", got, agreed)
	}
	var itemTerms, voucherTerms []byte
	if err := env.fixture.pool.QueryRow(env.ctx, `SELECT terms_snapshot FROM order_items WHERE order_id = $1`, order.ID).Scan(&itemTerms); err != nil {
		t.Fatal(err)
	}
	if err := env.fixture.pool.QueryRow(env.ctx, `SELECT oi.terms_snapshot FROM vouchers v JOIN order_items oi ON oi.id = v.order_item_id WHERE oi.order_id = $1`, order.ID).Scan(&voucherTerms); err != nil {
		t.Fatal(err)
	}
	if string(itemTerms) != string(agreedTerms) || string(voucherTerms) != string(agreedTerms) {
		t.Fatalf("terms snapshots changed: item=%s voucher=%s want=%s", itemTerms, voucherTerms, agreedTerms)
	}
}
