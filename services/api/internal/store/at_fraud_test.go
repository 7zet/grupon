package store_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/grupon/grupon/services/api/internal/domain/commerce"
	"github.com/grupon/grupon/services/api/internal/domain/redemption"
	"github.com/grupon/grupon/services/api/internal/payments"
	"github.com/grupon/grupon/services/api/internal/store"
)

// AT-14: one staff member scans eight vouchers within a minute. The burst rule records a fraud signal, holds
// the merchant's payouts, flags the vouchers scanned in the burst, and refuses a flagged voucher afterwards.
func TestAT14ScanBurstRecordsSignalHoldsPayoutsAndFlagsVouchers(t *testing.T) {
	env := newATEnv(t, 9)

	// One unit per buyer: the deal limits how many a single buyer may take.
	var voucherIDs []uuid.UUID
	for i := 0; i < 8; i++ {
		voucherIDs = append(voucherIDs, env.sellOneVoucher(t, env.newBuyer(t)))
	}
	merchantID, locationID := env.merchantAndLocation(t)

	repo := store.NewRedemptionRepository(env.fixture.pool, store.NewCommerceRepository(env.fixture.pool), store.NewCatalogRepository(env.fixture.pool))
	staff := env.buyer // any user can act as the staff member in this test
	now := time.Now()
	for i, voucherID := range voucherIDs {
		audit := redemption.AuditEntry{ActorID: staff, Role: "merchant_cashier", VoucherID: voucherID, Action: "voucher_redeemed"}
		ok, err := repo.UseVoucher(env.ctx, merchantID, voucherID, locationID, now, audit, nil)
		if err != nil || !ok {
			t.Fatalf("scan %d: ok=%v err=%v", i+1, ok, err)
		}
	}

	if got := env.count(t, `SELECT count(*) FROM fraud_signals WHERE rule = 'redemption_burst' AND subject_user_id = $1`, staff); got == 0 {
		t.Fatal("no redemption_burst fraud signal was recorded")
	}
	if got := env.count(t, `SELECT count(*) FROM merchant_payout_holds WHERE merchant_id = $1`, merchantID); got != 1 {
		t.Fatal("merchant payouts were not held")
	}
	if got := env.count(t, `SELECT count(*) FROM vouchers WHERE id = ANY($1) AND fraud_hold`, voucherIDs); got == 0 {
		t.Fatal("no voucher from the burst was flagged fraud_hold")
	}

	// A voucher that is flagged and still unused cannot be scanned.
	heldID := env.sellOneVoucher(t, env.newBuyer(t))
	if _, err := env.fixture.pool.Exec(env.ctx, `UPDATE vouchers SET fraud_hold = true WHERE id = $1`, heldID); err != nil {
		t.Fatal(err)
	}
	audit := redemption.AuditEntry{ActorID: staff, Role: "merchant_cashier", VoucherID: heldID, Action: "voucher_redeemed"}
	ok, err := repo.UseVoucher(env.ctx, merchantID, heldID, locationID, time.Now(), audit, nil)
	if ok || !errors.Is(err, redemption.ErrOnHold) {
		t.Fatalf("held voucher: ok=%v err=%v, want ErrOnHold", ok, err)
	}
}

// newBuyer creates a fresh customer account.
func (e atEnv) newBuyer(t *testing.T) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := e.fixture.pool.QueryRow(e.ctx, `
		INSERT INTO users (email, name) VALUES ($1, 'AT buyer') RETURNING id`, "at-buyer-"+uuid.NewString()+"@test.local").Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// sellOneVoucher buys one unit for the buyer, pays it, and returns the voucher id.
func (e atEnv) sellOneVoucher(t *testing.T, buyer uuid.UUID) uuid.UUID {
	t.Helper()
	order, _, err := e.service.CreateOrder(e.ctx, buyer, commerceOrderInput(e.optionID, 1))
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	session, err := e.service.PayOrder(e.ctx, buyer, order.ID, e.mock.Name())
	if err != nil {
		t.Fatalf("pay order: %v", err)
	}
	if err := e.service.CompleteMockPayment(e.ctx, session.PaymentID, payments.OutcomeSucceeded); err != nil {
		t.Fatalf("complete payment: %v", err)
	}
	var id uuid.UUID
	if err := e.fixture.pool.QueryRow(e.ctx, `
		SELECT v.id FROM vouchers v JOIN order_items oi ON oi.id = v.order_item_id WHERE oi.order_id = $1`, order.ID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// merchantAndLocation returns the merchant that sells the test option and one of its branches.
func (e atEnv) merchantAndLocation(t *testing.T) (uuid.UUID, uuid.UUID) {
	t.Helper()
	var merchantID, locationID uuid.UUID
	if err := e.fixture.pool.QueryRow(e.ctx, `
		SELECT d.merchant_id FROM deal_options o JOIN deals d ON d.id = o.deal_id WHERE o.id = $1`, e.optionID).Scan(&merchantID); err != nil {
		t.Fatal(err)
	}
	if err := e.fixture.pool.QueryRow(e.ctx, `
		SELECT id FROM merchant_locations WHERE merchant_id = $1 ORDER BY created_at LIMIT 1`, merchantID).Scan(&locationID); err != nil {
		t.Fatal(err)
	}
	return merchantID, locationID
}

// commerceOrderInput is a one-line order with a fresh idempotency key.
func commerceOrderInput(optionID uuid.UUID, quantity int) commerce.OrderInput {
	return commerce.OrderInput{
		IdempotencyKey: uuid.NewString(),
		Items:          []commerce.OrderItemInput{{OptionID: optionID, Quantity: quantity}},
	}
}
