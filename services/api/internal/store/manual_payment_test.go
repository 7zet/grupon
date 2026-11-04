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

func TestManualPaymentConfirmationIsIdempotentAndAudited(t *testing.T) {
	e := newATEnv(t, 4)
	signer, err := commerce.NewQRSigner("test-qr-secret-with-at-least-32-bytes!!")
	if err != nil {
		t.Fatal(err)
	}
	manual := &payments.Manual{Instructions: "Transfer to pilot account"}
	service, err := commerce.NewService(commerce.Dependencies{Repository: store.NewCommerceRepository(e.fixture.pool), Deals: store.NewCatalogRepository(e.fixture.pool), Manual: manual, ManualHold: 24 * time.Hour, Signer: signer, Mailer: mail.NewLogSender(slog.New(slog.NewTextHandler(io.Discard, nil))), Now: time.Now})
	if err != nil {
		t.Fatal(err)
	}
	order, _, err := service.CreateOrder(context.Background(), e.buyer, commerce.OrderInput{IdempotencyKey: uuid.NewString(), Items: []commerce.OrderItemInput{{OptionID: e.optionID, Quantity: 2}}})
	if err != nil {
		t.Fatal(err)
	}
	session, err := service.PayOrder(context.Background(), e.buyer, order.ID, "manual")
	if err != nil {
		t.Fatal(err)
	}
	if session.Instructions == "" {
		t.Fatal("manual instructions missing")
	}
	actor := uuid.New()
	if err := service.ConfirmManualPayment(context.Background(), session.PaymentID, actor, "admin", "BANK-123", "matched statement"); err != nil {
		t.Fatal(err)
	}
	if err := service.ConfirmManualPayment(context.Background(), session.PaymentID, actor, "admin", "BANK-123", "matched statement"); err != nil {
		t.Fatal(err)
	}
	if got := e.count(t, `SELECT count(*) FROM vouchers WHERE order_item_id IN (SELECT id FROM order_items WHERE order_id=$1)`, order.ID); got != 2 {
		t.Fatalf("vouchers=%d want 2", got)
	}
	if got := e.count(t, `SELECT count(*) FROM journal_entries WHERE kind='payment_received' AND reference IN (SELECT id::text FROM order_items WHERE order_id=$1)`, order.ID); got != 2 {
		t.Fatalf("ledger=%d want 2", got)
	}
	if got := e.count(t, `SELECT count(*) FROM audit_logs WHERE entity_type='payment' AND action='manual_payment_confirmed'`); got != 1 {
		t.Fatalf("manual confirmation audits=%d want 1", got)
	}
}
