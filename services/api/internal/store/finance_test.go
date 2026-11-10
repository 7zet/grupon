package store_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dealo/dealo/services/api/internal/domain/finance"
	"github.com/dealo/dealo/services/api/internal/domain/ledger"
	"github.com/dealo/dealo/services/api/internal/domain/moderation"
	"github.com/dealo/dealo/services/api/internal/domain/outbox"
	"github.com/dealo/dealo/services/api/internal/domain/redemption"
	"github.com/dealo/dealo/services/api/internal/store"
	sqlc "github.com/dealo/dealo/services/api/internal/store/sqlc"
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
	if err := fixture.pool.QueryRow(ctx, `INSERT INTO order_items(order_id,deal_id,deal_option_id,title_snapshot,unit_price_snapshot,original_price_snapshot,terms_snapshot,quantity) SELECT $1,o.deal_id,o.id,o.title,$2,o.original_price,d.terms,1 FROM deal_options o JOIN deals d ON d.id=o.deal_id WHERE o.id=$3 RETURNING id`,
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
	var denied int
	if err := fixture.pool.QueryRow(ctx, `SELECT count(*) FROM audit_logs WHERE action='payout_approve_self_denied' AND entity_id=$1`, runID).Scan(&denied); err != nil || denied != 1 {
		t.Fatalf("self-approval audit rows = %d, err = %v; want 1", denied, err)
	}
	supportActorID := seededID(t, fixture, `SELECT id FROM users WHERE email='admin@local.test'`)
	if err := service.ApprovePayout(ctx, finance.Actor{UserID: supportActorID, Role: "support"}, runID, "BANK-1"); !errors.Is(err, finance.ErrForbidden) {
		t.Fatalf("wrong-role approval: got %v", err)
	}
	if err := fixture.pool.QueryRow(ctx, `SELECT count(*) FROM audit_logs WHERE action='payout_approve_denied' AND entity_id=$1`, runID).Scan(&denied); err != nil || denied != 1 {
		t.Fatalf("wrong-role approval audit rows = %d, err = %v; want 1", denied, err)
	}
	approverID := seededID(t, fixture, `SELECT id FROM users WHERE email='owner@local.test'`)
	if err := service.ApprovePayout(ctx, finance.Actor{UserID: approverID, Role: "finance"}, runID, "BANK-1"); err != nil {
		t.Fatalf("approve: %v", err)
	}
	if diff := unbalancedLines(t, fixture); diff != 0 {
		t.Fatalf("ledger unbalanced after payout: %d", diff)
	}
}

func TestReversalAfterPayoutBecomesMerchantReceivableAndOffsetsNextPayout(t *testing.T) {
	fixture := newCatalogFixture(t)
	ctx := context.Background()
	service := finance.NewService(store.NewFinanceRepository(fixture.pool), time.Now)
	preparer := finance.Actor{UserID: seededID(t, fixture, `SELECT id FROM users WHERE email='admin@local.test'`), Role: "finance"}
	approver := finance.Actor{UserID: seededID(t, fixture, `SELECT id FROM users WHERE email='owner@local.test'`), Role: "finance"}

	firstVoucher, _, merchantID, _ := freshVoucher(t, fixture, time.Now().Add(30*24*time.Hour))
	var firstRedemption uuid.UUID
	if err := fixture.pool.QueryRow(ctx, `INSERT INTO redemptions(voucher_id,merchant_id,merchant_location_id,staff_user_id,sequence,amount,commission,created_at)
		SELECT $1,$2,(SELECT id FROM merchant_locations WHERE merchant_id=$2 LIMIT 1),(SELECT id FROM users WHERE email='owner@local.test'),1,100000,15000,now()-interval '4 days' RETURNING id`,
		firstVoucher, merchantID).Scan(&firstRedemption); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.pool.Exec(ctx, `UPDATE vouchers SET redemptions_used=1, status='redeemed', redeemed_at=now(),
		redeemed_by_user_id=(SELECT id FROM users WHERE email='owner@local.test'),
		redeemed_location_id=(SELECT id FROM merchant_locations WHERE merchant_id=$2 LIMIT 1) WHERE id=$1`, firstVoucher, merchantID); err != nil {
		t.Fatal(err)
	}
	firstRun, err := service.PreparePayout(ctx, preparer, merchantID)
	if err != nil {
		t.Fatalf("prepare first payout: %v", err)
	}
	if err := service.ApprovePayout(ctx, approver, firstRun, "BANK-FIRST"); err != nil {
		t.Fatalf("approve first payout: %v", err)
	}

	redemptions := store.NewRedemptionRepository(fixture.pool, store.NewCommerceRepository(fixture.pool), store.NewCatalogRepository(fixture.pool))
	candidate, err := redemptions.ReversalCandidate(ctx, merchantID, firstVoucher)
	if err != nil || candidate.ID != firstRedemption || !candidate.PayoutSent {
		t.Fatalf("payout reversal candidate = %+v, err=%v", candidate, err)
	}
	if err := redemptions.ReverseUse(ctx, merchantID, candidate, "test reversal", redemption.AuditEntry{ActorID: preparer.UserID, Role: "support", VoucherID: firstVoucher, Action: "voucher_reversal"}); err != nil {
		t.Fatalf("reverse paid redemption: %v", err)
	}
	var receivable int64
	if err := fixture.pool.QueryRow(ctx, `SELECT COALESCE(SUM(CASE WHEN jl.direction='debit' THEN jl.amount ELSE -jl.amount END),0)
		FROM journal_lines jl JOIN ledger_accounts la ON la.id=jl.account_id WHERE la.code=$1`, ledger.MerchantReceivable+merchantID.String()).Scan(&receivable); err != nil || receivable != 85000 {
		t.Fatalf("merchant receivable=%d err=%v; want 85000", receivable, err)
	}

	secondVoucher, _, _, _ := freshVoucher(t, fixture, time.Now().Add(30*24*time.Hour))
	if _, err := fixture.pool.Exec(ctx, `INSERT INTO redemptions(voucher_id,merchant_id,merchant_location_id,staff_user_id,sequence,amount,commission,created_at)
		SELECT $1,$2,(SELECT id FROM merchant_locations WHERE merchant_id=$2 LIMIT 1),(SELECT id FROM users WHERE email='owner@local.test'),1,200000,30000,now()-interval '4 days'`, secondVoucher, merchantID); err != nil {
		t.Fatal(err)
	}
	secondRun, err := service.PreparePayout(ctx, preparer, merchantID)
	if err != nil {
		t.Fatalf("prepare offset payout: %v", err)
	}
	run, err := store.NewFinanceRepository(fixture.pool).PayoutRun(ctx, secondRun)
	if err != nil || run.Amount != 85000 || run.ReceivableOffset != 85000 {
		t.Fatalf("offset payout=%+v err=%v; want cash and offset 85000", run, err)
	}
	if err := service.ApprovePayout(ctx, approver, secondRun, "BANK-OFFSET"); err != nil {
		t.Fatalf("approve offset payout: %v", err)
	}
	if err := fixture.pool.QueryRow(ctx, `SELECT COALESCE(SUM(CASE WHEN jl.direction='debit' THEN jl.amount ELSE -jl.amount END),0)
		FROM journal_lines jl JOIN ledger_accounts la ON la.id=jl.account_id WHERE la.code=$1`, ledger.MerchantReceivable+merchantID.String()).Scan(&receivable); err != nil || receivable != 0 {
		t.Fatalf("merchant receivable after offset=%d err=%v; want 0", receivable, err)
	}
	if diff := unbalancedLines(t, fixture); diff != 0 {
		t.Fatalf("ledger unbalanced after paid reversal: %d", diff)
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

func TestPartialAndRecentlyExpiredVoucherRefundUsesRemainingShares(t *testing.T) {
	fixture := newCatalogFixture(t)
	ctx := context.Background()
	voucherID, buyerID, _, price := freshVoucher(t, fixture, time.Now().Add(30*24*time.Hour))
	if _, err := fixture.pool.Exec(ctx, `UPDATE order_items SET unit_price_snapshot=100001 WHERE id=(SELECT order_item_id FROM vouchers WHERE id=$1)`, voucherID); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.pool.Exec(ctx, `UPDATE vouchers SET redemptions_allowed=3, redemptions_used=1, status='partially_redeemed' WHERE id=$1`, voucherID); err != nil {
		t.Fatal(err)
	}
	service := finance.NewService(store.NewFinanceRepository(fixture.pool), time.Now)
	if _, err := service.RequestRefund(ctx, buyerID, voucherID, "one visit used"); err != nil {
		t.Fatalf("partial refund: %v", err)
	}
	var amount int64
	if err := fixture.pool.QueryRow(ctx, `SELECT amount FROM refunds WHERE voucher_id=$1`, voucherID).Scan(&amount); err != nil {
		t.Fatal(err)
	}
	if amount != 66668 { // 100001 split as 33333, 33333, 33335; one visit was used.
		t.Fatalf("partial refund amount=%d, want 66668 (original price %d)", amount, price)
	}
	var status string
	if err := fixture.pool.QueryRow(ctx, `SELECT status FROM vouchers WHERE id=$1`, voucherID).Scan(&status); err != nil || status != "refunded" {
		t.Fatalf("partial voucher status=%q err=%v", status, err)
	}

	expiredID, expiredBuyer, _, _ := freshVoucher(t, fixture, time.Now().Add(-24*time.Hour))
	if _, err := fixture.pool.Exec(ctx, `UPDATE vouchers SET status='expired' WHERE id=$1`, expiredID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.RequestRefund(ctx, expiredBuyer, expiredID, "recently expired"); err != nil {
		t.Fatalf("recent expired refund: %v", err)
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
		}, nil); err != nil {
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

func TestFinanceApprovesRefundOlderThanFourteenDays(t *testing.T) {
	fixture := newCatalogFixture(t)
	ctx := context.Background()
	voucherID, buyerID, _, _ := freshVoucher(t, fixture, time.Now().Add(30*24*time.Hour))
	if _, err := fixture.pool.Exec(ctx, `UPDATE orders SET paid_at = now() - interval '20 days'
		WHERE id = (SELECT oi.order_id FROM vouchers v JOIN order_items oi ON oi.id = v.order_item_id WHERE v.id = $1)`, voucherID); err != nil {
		t.Fatal(err)
	}
	service := finance.NewService(store.NewFinanceRepository(fixture.pool), time.Now)
	refundID, err := service.RequestRefund(ctx, buyerID, voucherID, "late request")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	var status string
	if err := fixture.pool.QueryRow(ctx, `SELECT status FROM refunds WHERE id=$1`, refundID).Scan(&status); err != nil || status != "requested" {
		t.Fatalf("refund status = %q, err = %v; want requested", status, err)
	}
	supportID := seededID(t, fixture, `SELECT id FROM users WHERE email='admin@local.test'`)
	support := finance.Actor{UserID: supportID, Role: "support"}
	if err := service.ApproveRefund(ctx, support, refundID); !errors.Is(err, finance.ErrForbidden) {
		t.Fatalf("support approval: got %v", err)
	}
	var denied int
	if err := fixture.pool.QueryRow(ctx, `SELECT count(*) FROM audit_logs WHERE action='refund_approve_denied' AND entity_id=$1`, refundID).Scan(&denied); err != nil || denied != 1 {
		t.Fatalf("support approval audit rows = %d, err = %v; want 1", denied, err)
	}
	if err := service.ProposeRefund(ctx, support, refundID); err != nil {
		t.Fatalf("support proposal: %v", err)
	}
	approver := finance.Actor{UserID: seededID(t, fixture, `SELECT id FROM users WHERE email='owner@local.test'`), Role: "finance"}
	if err := service.ApproveRefund(ctx, approver, refundID); err != nil {
		t.Fatalf("finance approval: %v", err)
	}
	if err := service.ApproveRefund(ctx, approver, refundID); !errors.Is(err, finance.ErrNotPending) {
		t.Fatalf("second approval: got %v", err)
	}
	if err := fixture.pool.QueryRow(ctx, `SELECT status FROM vouchers WHERE id=$1`, voucherID).Scan(&status); err != nil || status != "refunded" {
		t.Fatalf("voucher status = %q, err = %v", status, err)
	}
	if ok, err := service.Reconcile(ctx); err != nil || !ok {
		t.Fatalf("reconcile after approval: ok=%v err=%v", ok, err)
	}
	if diff := unbalancedLines(t, fixture); diff != 0 {
		t.Fatalf("ledger unbalanced after refund approval: %d", diff)
	}
}

func openTestDispute(t *testing.T, fixture catalogFixture) (uuid.UUID, uuid.UUID, uuid.UUID, *finance.Service) {
	t.Helper()
	ctx := context.Background()
	voucherID, buyerID, merchantID, price := freshVoucher(t, fixture, time.Now().Add(30*24*time.Hour))
	var redemptionID uuid.UUID
	if err := fixture.pool.QueryRow(ctx, `INSERT INTO redemptions(voucher_id,merchant_id,merchant_location_id,staff_user_id,sequence,amount,commission,created_at)
		SELECT $1,$2,(SELECT id FROM merchant_locations WHERE merchant_id=$2 LIMIT 1),(SELECT id FROM users WHERE email='owner@local.test'),1,$3,0,now()-interval '4 days' RETURNING id`,
		voucherID, merchantID, price).Scan(&redemptionID); err != nil {
		t.Fatal(err)
	}
	service := finance.NewService(store.NewFinanceRepository(fixture.pool), time.Now)
	if err := service.OpenDispute(ctx, buyerID, voucherID, "not my redemption"); err != nil {
		t.Fatalf("open dispute: %v", err)
	}
	var disputeID uuid.UUID
	if err := fixture.pool.QueryRow(ctx, `SELECT id FROM disputes WHERE redemption_id=$1`, redemptionID).Scan(&disputeID); err != nil {
		t.Fatal(err)
	}
	return disputeID, redemptionID, merchantID, service
}

func disputeStatus(t *testing.T, fixture catalogFixture, disputeID uuid.UUID) string {
	t.Helper()
	var status string
	if err := fixture.pool.QueryRow(context.Background(), `SELECT status FROM disputes WHERE id=$1`, disputeID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	return status
}

func TestDisputeAllowedTransitionsAuditAndPayoutHold(t *testing.T) {
	fixture := newCatalogFixture(t)
	ctx := context.Background()
	disputeID, redemptionID, merchantID, service := openTestDispute(t, fixture)
	support := finance.Actor{UserID: seededID(t, fixture, `SELECT id FROM users WHERE email='admin@local.test'`), Role: "support"}
	approver := finance.Actor{UserID: seededID(t, fixture, `SELECT id FROM users WHERE email='owner@local.test'`), Role: "finance"}
	repo := store.NewFinanceRepository(fixture.pool)

	assertHeld := func(wantHeld bool) {
		items, err := repo.PayableRedemptions(ctx, merchantID, time.Now().Add(-3*24*time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, item := range items {
			found = found || item.ID == redemptionID
		}
		if found == wantHeld {
			t.Fatalf("payable=%v, want held=%v", found, wantHeld)
		}
	}

	assertHeld(true)
	if err := service.RequestDisputeEvidence(ctx, support, disputeID); err != nil {
		t.Fatal(err)
	}
	if got := disputeStatus(t, fixture, disputeID); got != "awaiting_evidence" {
		t.Fatalf("status=%s", got)
	}
	assertHeld(true)
	if err := service.StartDisputeReview(ctx, support, disputeID); err != nil {
		t.Fatal(err)
	}
	if got := disputeStatus(t, fixture, disputeID); got != "under_review" {
		t.Fatalf("status=%s", got)
	}
	assertHeld(true)
	if err := service.ResolveDispute(ctx, approver, disputeID, "merchant"); err != nil {
		t.Fatal(err)
	}
	if got := disputeStatus(t, fixture, disputeID); got != "resolved_merchant" {
		t.Fatalf("status=%s", got)
	}
	assertHeld(false)
	if err := service.CloseDispute(ctx, support, disputeID); err != nil {
		t.Fatal(err)
	}
	if got := disputeStatus(t, fixture, disputeID); got != "closed" {
		t.Fatalf("status=%s", got)
	}
	var audits int
	if err := fixture.pool.QueryRow(ctx, `SELECT count(*) FROM audit_logs WHERE entity_type='dispute' AND entity_id=$1`, disputeID).Scan(&audits); err != nil || audits != 4 {
		t.Fatalf("audits=%d err=%v", audits, err)
	}
}

func TestDisputeCustomerResolutionKeepsRefundBehavior(t *testing.T) {
	fixture := newCatalogFixture(t)
	ctx := context.Background()
	disputeID, _, _, service := openTestDispute(t, fixture)
	support := finance.Actor{UserID: seededID(t, fixture, `SELECT id FROM users WHERE email='admin@local.test'`), Role: "support"}
	approver := finance.Actor{UserID: seededID(t, fixture, `SELECT id FROM users WHERE email='owner@local.test'`), Role: "finance"}
	if err := service.RequestDisputeEvidence(ctx, support, disputeID); err != nil {
		t.Fatal(err)
	}
	if err := service.StartDisputeReview(ctx, support, disputeID); err != nil {
		t.Fatal(err)
	}
	if err := service.ResolveDispute(ctx, approver, disputeID, "customer"); err != nil {
		t.Fatal(err)
	}
	var refunds int
	if err := fixture.pool.QueryRow(ctx, `SELECT count(*) FROM refunds r JOIN disputes d ON d.id=$1 JOIN redemptions rd ON rd.id=d.redemption_id WHERE r.voucher_id=rd.voucher_id AND r.status='requested'`, disputeID).Scan(&refunds); err != nil || refunds != 1 {
		t.Fatalf("refunds=%d err=%v", refunds, err)
	}
	var disputeRefundID uuid.UUID
	if err := fixture.pool.QueryRow(ctx, `SELECT r.id FROM refunds r JOIN disputes d ON d.id=$1 JOIN redemptions rd ON rd.id=d.redemption_id WHERE r.voucher_id=rd.voucher_id`, disputeID).Scan(&disputeRefundID); err != nil {
		t.Fatal(err)
	}
	if err := service.ProposeRefund(ctx, support, disputeRefundID); err != nil {
		t.Fatalf("support proposal: %v", err)
	}
	otherFinance := finance.Actor{UserID: seededID(t, fixture, `SELECT id FROM users WHERE email='user@local.test'`), Role: "finance"}
	if err := service.ApproveRefund(ctx, otherFinance, disputeRefundID); err != nil {
		t.Fatalf("finance approval: %v", err)
	}
	if err := service.CloseDispute(ctx, finance.Actor{Role: "system"}, disputeID); err != nil {
		t.Fatal(err)
	}
}

func TestDisputeForbiddenTransitionAndRBAC(t *testing.T) {
	fixture := newCatalogFixture(t)
	ctx := context.Background()
	disputeID, _, _, service := openTestDispute(t, fixture)
	support := finance.Actor{UserID: seededID(t, fixture, `SELECT id FROM users WHERE email='admin@local.test'`), Role: "support"}
	wrong := finance.Actor{UserID: support.UserID, Role: "user"}
	if err := service.StartDisputeReview(ctx, support, disputeID); !errors.Is(err, finance.ErrInvalidTransition) {
		t.Fatalf("forbidden transition: %v", err)
	}
	if err := service.RequestDisputeEvidence(ctx, wrong, disputeID); !errors.Is(err, finance.ErrForbidden) {
		t.Fatalf("request evidence RBAC: %v", err)
	}
	if err := service.ResolveDispute(ctx, support, disputeID, "merchant"); !errors.Is(err, finance.ErrForbidden) {
		t.Fatalf("resolve RBAC: %v", err)
	}
	if err := service.CloseDispute(ctx, finance.Actor{UserID: support.UserID, Role: "finance"}, disputeID); !errors.Is(err, finance.ErrForbidden) {
		t.Fatalf("close RBAC: %v", err)
	}
}

func TestOutboxPublishesEachEventOnce(t *testing.T) {
	fixture := newCatalogFixture(t)
	ctx := context.Background()
	freshVoucher(t, fixture, time.Now().Add(30*24*time.Hour))
	var published []string
	relay := outbox.NewRelay(store.NewOutboxRepository(fixture.pool), func(_ context.Context, topic string, _ []byte) error {
		published = append(published, topic)
		return nil
	})
	first, err := relay.Run(ctx, 1000)
	if err != nil || first < 1 {
		t.Fatalf("first run: published=%d err=%v", first, err)
	}
	if len(published) != first {
		t.Fatalf("publisher saw %d events, relay reported %d", len(published), first)
	}
	second, err := relay.Run(ctx, 1000)
	if err != nil || second != 0 {
		t.Fatalf("second run: published=%d err=%v; want 0", second, err)
	}
}
