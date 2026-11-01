package store_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/grupon/grupon/services/api/internal/domain/finance"
	"github.com/grupon/grupon/services/api/internal/domain/ledger"
	"github.com/grupon/grupon/services/api/internal/domain/moderation"
	"github.com/grupon/grupon/services/api/internal/domain/redemption"
	"github.com/grupon/grupon/services/api/internal/store"
	sqlc "github.com/grupon/grupon/services/api/internal/store/sqlc"
	"github.com/jackc/pgx/v5"
)

// freshVoucher creates a paid, unused voucher for the first seeded buyer on the first seeded option.
func freshVoucher(t *testing.T, fixture catalogFixture, validUntil time.Time) (voucherID, buyerID, merchantID uuid.UUID, price int64) {
	t.Helper()
	ctx := context.Background()
	buyerID = seededID(t, fixture, `SELECT id FROM users WHERE email = 'user@local.test'`)
	var optionID, dealID, locationID string
	if err := fixture.pool.QueryRow(ctx, `
		SELECT d.id, o.id, m.id::text, (SELECT id FROM merchant_locations ml WHERE ml.merchant_id = m.id LIMIT 1)
		FROM deal_options o JOIN deals d ON d.id = o.deal_id JOIN merchants m ON m.id = d.merchant_id
		WHERE d.status = 'active' AND o.is_active ORDER BY o.id LIMIT 1`).Scan(&dealID, &optionID, &merchantID, &locationID); err != nil {
		t.Fatalf("option: %v", err)
	}
	if err := fixture.pool.QueryRow(ctx, `SELECT deal_price FROM deal_options WHERE id = $1`, optionID).Scan(&price); err != nil {
		t.Fatal(err)
	}
	var orderID, itemID string
	if err := fixture.pool.QueryRow(ctx, `INSERT INTO orders(number,user_id,status,subtotal,total,idempotency_key,paid_at) VALUES('GR-T-' || nextval('order_number_seq'),$1,'paid',$2,$2,$3,now()) RETURNING id`,
		buyerID, price, uuid.NewString()).Scan(&orderID); err != nil {
		t.Fatalf("order: %v", err)
	}
	if err := fixture.pool.QueryRow(ctx, `INSERT INTO order_items(order_id,deal_id,deal_option_id,title_snapshot,unit_price_snapshot,original_price_snapshot,quantity) SELECT $1,deal_id,id,title,$2,original_price,1 FROM deal_options WHERE id=$3 RETURNING id`,
		orderID, price, optionID).Scan(&itemID); err != nil {
		t.Fatalf("item: %v", err)
	}
	if err := fixture.pool.QueryRow(ctx, `INSERT INTO vouchers(order_item_id,user_id,deal_id,deal_option_id,merchant_id,code,qr_token,status,valid_from,valid_until)
		SELECT $1,$2,deal_id,$3,$4,$5,$6,'active',$7::timestamptz - interval '30 days',$7 FROM order_items WHERE id=$1 RETURNING id`,
		itemID, buyerID, optionID, merchantID, testCode(), uuid.NewString(), validUntil).Scan(&voucherID); err != nil {
		t.Fatalf("voucher: %v", err)
	}
	// Post the payment the way the commerce flow does, so the ledger knows about this voucher.
	if err := postPayment(t, fixture, itemID, merchantID, price); err != nil {
		t.Fatal(err)
	}
	return voucherID, buyerID, merchantID, price
}

func postPayment(t *testing.T, fixture catalogFixture, itemID string, merchantID uuid.UUID, price int64) error {
	t.Helper()
	tx, err := fixture.pool.Begin(context.Background())
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background()) //nolint:errcheck
	_, err = store.PostEntry(context.Background(), sqlcFrom(tx), paymentEntry(itemID, merchantID.String(), price))
	if err != nil {
		return err
	}
	return tx.Commit(context.Background())
}

func unbalancedLines(t *testing.T, fixture catalogFixture) int64 {
	t.Helper()
	var diff int64
	if err := fixture.pool.QueryRow(context.Background(), `
		SELECT COALESCE(SUM(CASE WHEN direction='debit' THEN amount ELSE -amount END),0) FROM journal_lines`).Scan(&diff); err != nil {
		t.Fatal(err)
	}
	return diff
}

func TestLedgerStaysBalancedAndReconciles(t *testing.T) {
	fixture := newCatalogFixture(t)
	if diff := unbalancedLines(t, fixture); diff != 0 {
		t.Fatalf("ledger debits and credits differ by %d after seeding", diff)
	}
	repo := store.NewFinanceRepository(fixture.pool)
	service := finance.NewService(repo, time.Now)
	ok, err := service.Reconcile(context.Background())
	if err != nil || !ok {
		t.Fatalf("reconcile after seed: ok=%v err=%v", ok, err)
	}
}

func TestPayoutNeedsADifferentApprover(t *testing.T) {
	fixture := newCatalogFixture(t)
	ctx := context.Background()
	voucherID, _, merchantID, _ := freshVoucher(t, fixture, time.Now().Add(30*24*time.Hour))
	_ = voucherID
	// Redeem it a few days ago so it is past the dispute window.
	if _, err := fixture.pool.Exec(ctx, `INSERT INTO redemptions(voucher_id,merchant_id,merchant_location_id,staff_user_id,sequence,amount,commission,created_at)
		SELECT $1,$2,(SELECT id FROM merchant_locations WHERE merchant_id=$2 LIMIT 1),(SELECT id FROM users WHERE email='owner@local.test'),1,100000,15000,now()-interval '4 days'`,
		voucherID, merchantID); err != nil {
		t.Fatal(err)
	}
	service := finance.NewService(store.NewFinanceRepository(fixture.pool), time.Now)
	preparer := finance.Actor{UserID: seededID(t, fixture, `SELECT id FROM users WHERE email='admin@local.test'`), Role: "finance"}
	runID, err := service.PreparePayout(ctx, preparer, merchantID)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if err := service.ApprovePayout(ctx, preparer, runID, "BANK-1"); !errors.Is(err, finance.ErrSelfApproval) {
		t.Fatalf("self approval: got %v", err)
	}
	approverID := seededID(t, fixture, `SELECT id FROM users WHERE email='owner@local.test'`)
	if err := service.ApprovePayout(ctx, finance.Actor{UserID: approverID, Role: "finance"}, runID, "BANK-1"); err != nil {
		t.Fatalf("approve: %v", err)
	}
	if diff := unbalancedLines(t, fixture); diff != 0 {
		t.Fatalf("ledger unbalanced after payout: %d", diff)
	}
}

func TestBreakageMovesExpiredValue(t *testing.T) {
	fixture := newCatalogFixture(t)
	voucherID, _, _, price := freshVoucher(t, fixture, time.Now().Add(-100*24*time.Hour))
	service := finance.NewService(store.NewFinanceRepository(fixture.pool), time.Now)
	moved, err := service.RunBreakage(context.Background(), 100)
	if err != nil || moved < 1 {
		t.Fatalf("breakage: moved=%d err=%v", moved, err)
	}
	var status string
	if err := fixture.pool.QueryRow(context.Background(), `SELECT status FROM vouchers WHERE id=$1`, voucherID).Scan(&status); err != nil || status != "expired" {
		t.Fatalf("status = %q, err = %v (price %d)", status, err, price)
	}
	ok, err := service.Reconcile(context.Background())
	if err != nil || !ok {
		t.Fatalf("reconcile after breakage: ok=%v err=%v", ok, err)
	}
}

func TestUnusedVoucherRefundsAutomaticallyWithinFourteenDays(t *testing.T) {
	fixture := newCatalogFixture(t)
	voucherID, buyerID, _, _ := freshVoucher(t, fixture, time.Now().Add(30*24*time.Hour))
	service := finance.NewService(store.NewFinanceRepository(fixture.pool), time.Now)
	if _, err := service.RequestRefund(context.Background(), buyerID, voucherID, "changed my mind"); err != nil {
		t.Fatalf("refund: %v", err)
	}
	var status string
	if err := fixture.pool.QueryRow(context.Background(), `SELECT status FROM vouchers WHERE id=$1`, voucherID).Scan(&status); err != nil || status != "refunded" {
		t.Fatalf("voucher status = %q, err = %v", status, err)
	}
	ok, err := service.Reconcile(context.Background())
	if err != nil || !ok {
		t.Fatalf("reconcile after refund: ok=%v err=%v", ok, err)
	}
}

func TestRedemptionBurstHoldsMerchantPayouts(t *testing.T) {
	fixture := newCatalogFixture(t)
	ctx := context.Background()
	cashierID := seededID(t, fixture, `SELECT id FROM users WHERE email='cashier@local.test'`)
	var merchantID uuid.UUID
	var locationID uuid.UUID
	merchantID = seededID(t, fixture, `SELECT merchant_id FROM role_assignments WHERE user_id = (SELECT id FROM users WHERE email='owner@local.test') AND merchant_id IS NOT NULL LIMIT 1`)
	locationID = seededID(t, fixture, `SELECT id FROM merchant_locations WHERE merchant_id=$1 LIMIT 1`, merchantID)
	repo := store.NewRedemptionRepository(fixture.pool, store.NewCommerceRepository(fixture.pool), store.NewCatalogRepository(fixture.pool))
	for i := 0; i < 6; i++ {
		voucherID, _, _, _ := freshVoucher(t, fixture, time.Now().Add(30*24*time.Hour))
		if _, err := repo.Redeem(ctx, merchantID, voucherID, locationID, time.Now(), redemption.AuditEntry{
			ActorID: cashierID, Role: "merchant_cashier", VoucherID: voucherID, Action: "voucher_redeemed",
		}); err != nil {
			t.Fatalf("redeem %d: %v", i, err)
		}
	}
	var holds int
	if err := fixture.pool.QueryRow(ctx, `SELECT count(*) FROM merchant_payout_holds WHERE merchant_id=$1`, merchantID).Scan(&holds); err != nil || holds != 1 {
		t.Fatalf("holds = %d, err = %v", holds, err)
	}
	var signals int
	if err := fixture.pool.QueryRow(ctx, `SELECT count(*) FROM fraud_signals WHERE merchant_id=$1 AND rule='redemption_burst'`, merchantID).Scan(&signals); err != nil || signals < 1 {
		t.Fatalf("fraud signals = %d, err = %v", signals, err)
	}
	service := finance.NewService(store.NewFinanceRepository(fixture.pool), time.Now)
	_, err := service.PreparePayout(ctx, finance.Actor{UserID: cashierID, Role: "finance"}, merchantID)
	if !errors.Is(err, finance.ErrMerchantOnHold) {
		t.Fatalf("payout while on hold: got %v", err)
	}
	_ = moderation.ErrNotFound
}

func sqlcFrom(tx pgx.Tx) *sqlc.Queries { return sqlc.New(tx) }

func paymentEntry(itemID, merchantID string, price int64) ledger.Entry {
	return ledger.PaymentReceived(itemID, merchantID, price)
}

// testCode returns a voucher code in the XXXX-XXXX form, unique enough for a test database.
func testCode() string {
	raw := strings.ToUpper(strings.ReplaceAll(uuid.NewString(), "-", ""))
	return raw[0:4] + "-" + raw[4:8]
}
