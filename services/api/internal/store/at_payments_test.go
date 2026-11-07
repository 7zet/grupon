package store_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/grupon/grupon/services/api/internal/domain/commerce"
	"github.com/grupon/grupon/services/api/internal/payments"
	"github.com/grupon/grupon/services/api/internal/platform/mail"
	"github.com/grupon/grupon/services/api/internal/store"
)

// atEnv is one sellable option with a known stock, a buyer, and a commerce service that can take mock payments.
type atEnv struct {
	fixture  catalogFixture
	ctx      context.Context
	optionID uuid.UUID
	buyer    uuid.UUID
	service  *commerce.Service
	mock     *payments.Mock
}

func newATEnv(t *testing.T, sellable int) atEnv {
	t.Helper()
	fixture := newCatalogFixture(t)
	ctx := context.Background()

	var optionID uuid.UUID
	if err := fixture.pool.QueryRow(ctx, `
		SELECT o.id FROM deal_options o
		JOIN deals d ON d.id = o.deal_id
		JOIN merchants m ON m.id = d.merchant_id
		WHERE d.status = 'active' AND d.ends_at > now()
		  AND o.is_active AND m.review_status = 'approved'
		LIMIT 1`).Scan(&optionID); err != nil {
		t.Fatal(err)
	}
	// Exactly `sellable` units are free for this test, whatever the seed sold or reserved.
	if _, err := fixture.pool.Exec(ctx, `
		UPDATE deal_options SET stock_total = stock_sold + stock_reserved + $2 WHERE id = $1`, optionID, sellable); err != nil {
		t.Fatal(err)
	}

	var buyer uuid.UUID
	if err := fixture.pool.QueryRow(ctx, `
		INSERT INTO users (email, name) VALUES ($1, 'AT buyer') RETURNING id`, "at-buyer-"+uuid.NewString()+"@test.local").Scan(&buyer); err != nil {
		t.Fatal(err)
	}

	mock, err := payments.NewMock("at-mock-webhook-secret-with-32-bytes!!")
	if err != nil {
		t.Fatal(err)
	}
	signer, err := commerce.NewQRSigner("test-qr-secret-with-at-least-32-bytes!!")
	if err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service, err := commerce.NewService(commerce.Dependencies{
		Repository: store.NewCommerceRepository(fixture.pool),
		Deals:      store.NewCatalogRepository(fixture.pool),
		Providers:  []payments.Provider{mock},
		Mock:       mock,
		Signer:     signer,
		Mailer:     mail.NewLogSender(logger),
		Logger:     logger,
		Now:        time.Now,
	})
	if err != nil {
		t.Fatal(err)
	}
	return atEnv{fixture: fixture, ctx: ctx, optionID: optionID, buyer: buyer, service: service, mock: mock}
}

func (e atEnv) order(t *testing.T, quantity int) commerce.Order {
	t.Helper()
	order, _, err := e.service.CreateOrder(e.ctx, e.buyer, commerce.OrderInput{
		IdempotencyKey: uuid.NewString(),
		Items:          []commerce.OrderItemInput{{OptionID: e.optionID, Quantity: quantity}},
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	return order
}

func (e atEnv) count(t *testing.T, query string, args ...any) int {
	t.Helper()
	var n int
	if err := e.fixture.pool.QueryRow(e.ctx, query, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func (e atEnv) scalarText(t *testing.T, query string, args ...any) string {
	t.Helper()
	var s string
	if err := e.fixture.pool.QueryRow(e.ctx, query, args...).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

// AT-01: the same success webhook arrives five times. Exactly one voucher set is issued and the order is paid once.
func TestAT01DuplicateSuccessWebhookIssuesOneVoucherSet(t *testing.T) {
	env := newATEnv(t, 5)
	order := env.order(t, 1)

	session, err := env.service.PayOrder(env.ctx, env.buyer, order.ID, env.mock.Name())
	if err != nil {
		t.Fatalf("pay order: %v", err)
	}
	providerID := env.scalarText(t, `SELECT provider_payment_id FROM payments WHERE id = $1`, session.PaymentID)
	body, signature, err := env.mock.SignedWebhook(providerID, payments.OutcomeSucceeded)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		if err := env.service.HandleWebhook(env.ctx, env.mock.Name(), body, signature); err != nil {
			t.Fatalf("webhook delivery %d: %v", i+1, err)
		}
	}

	if got := env.count(t, `SELECT count(*) FROM vouchers v JOIN order_items oi ON oi.id = v.order_item_id WHERE oi.order_id = $1`, order.ID); got != 1 {
		t.Fatalf("vouchers = %d, want 1", got)
	}
	if got := env.scalarText(t, `SELECT status::text FROM orders WHERE id = $1`, order.ID); got != "paid" {
		t.Fatalf("order status = %s, want paid", got)
	}
	if got := env.scalarText(t, `SELECT status::text FROM payments WHERE id = $1`, session.PaymentID); got != "succeeded" {
		t.Fatalf("payment status = %s, want succeeded", got)
	}
	if got := env.count(t, `SELECT count(*) FROM journal_entries WHERE reference = (SELECT id::text FROM order_items WHERE order_id = $1)`, order.ID); got != 1 {
		t.Fatalf("journal entries = %d, want exactly 1", got)
	}
}

// AT-07: the payment fails, the hold runs out. Stock comes back, no voucher exists, and the order is expired.
func TestAT07FailedPaymentAndExpiredHoldReturnStockAndIssueNoVoucher(t *testing.T) {
	env := newATEnv(t, 3)
	reservedBefore := env.count(t, `SELECT stock_reserved FROM deal_options WHERE id = $1`, env.optionID)
	order := env.order(t, 2)

	session, err := env.service.PayOrder(env.ctx, env.buyer, order.ID, env.mock.Name())
	if err != nil {
		t.Fatalf("pay order: %v", err)
	}
	if err := env.service.CompleteMockPayment(env.ctx, session.PaymentID, payments.OutcomeFailed); err != nil {
		t.Fatalf("fail payment: %v", err)
	}
	// The 15-minute hold is over.
	if _, err := env.fixture.pool.Exec(env.ctx, `
		UPDATE stock_reservations SET expires_at = now() - interval '1 minute' WHERE order_id = $1`, order.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := env.service.ReleaseExpiredReservations(env.ctx, 100); err != nil {
		t.Fatalf("release expired holds: %v", err)
	}

	if got := env.count(t, `SELECT stock_reserved FROM deal_options WHERE id = $1`, env.optionID); got != reservedBefore {
		t.Fatalf("stock_reserved = %d, want %d (stock returned)", got, reservedBefore)
	}
	if got := env.count(t, `SELECT count(*) FROM vouchers v JOIN order_items oi ON oi.id = v.order_item_id WHERE oi.order_id = $1`, order.ID); got != 0 {
		t.Fatalf("vouchers = %d, want 0", got)
	}
	if got := env.scalarText(t, `SELECT status::text FROM orders WHERE id = $1`, order.ID); got != "expired" {
		t.Fatalf("order status = %s, want expired", got)
	}
}

// AT-13: the hold runs out and the order expires. The provider's success arrives later. The system refunds the
// payment, no voucher is issued, and the refund is recorded against the payment.
func TestAT13LatePaymentIsRefundedAndRecorded(t *testing.T) {
	env := newATEnv(t, 1)
	order := env.order(t, 1)
	session, err := env.service.PayOrder(env.ctx, env.buyer, order.ID, env.mock.Name())
	if err != nil {
		t.Fatalf("pay order: %v", err)
	}
	if _, err := env.fixture.pool.Exec(env.ctx, `
		UPDATE stock_reservations SET expires_at = now() - interval '1 minute' WHERE order_id = $1`, order.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := env.service.ReleaseExpiredReservations(env.ctx, 100); err != nil {
		t.Fatalf("release expired holds: %v", err)
	}
	if got := env.scalarText(t, `SELECT status::text FROM orders WHERE id = $1`, order.ID); got != "expired" {
		t.Fatalf("order status = %s, want expired before the late payment", got)
	}

	providerID := env.scalarText(t, `SELECT provider_payment_id FROM payments WHERE id = $1`, session.PaymentID)
	body, signature, err := env.mock.SignedWebhook(providerID, payments.OutcomeSucceeded)
	if err != nil {
		t.Fatal(err)
	}
	if err := env.service.HandleWebhook(env.ctx, env.mock.Name(), body, signature); err != nil {
		t.Fatalf("late success webhook: %v", err)
	}

	if got := env.count(t, `SELECT count(*) FROM refunds WHERE payment_id = $1 AND status = 'succeeded'`, session.PaymentID); got != 1 {
		t.Fatalf("refund rows = %d, want 1", got)
	}
	if got := env.count(t, `SELECT count(*) FROM vouchers v JOIN order_items oi ON oi.id = v.order_item_id WHERE oi.order_id = $1`, order.ID); got != 0 {
		t.Fatalf("vouchers = %d, want 0", got)
	}
	if got := env.scalarText(t, `SELECT status::text FROM orders WHERE id = $1`, order.ID); got != "expired" {
		t.Fatalf("order status = %s, want still expired", got)
	}
}
