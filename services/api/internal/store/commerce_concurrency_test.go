package store_test

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/grupon/grupon/services/api/internal/domain/commerce"
	"github.com/grupon/grupon/services/api/internal/platform/mail"
	"github.com/grupon/grupon/services/api/internal/store"
)

// TestOverSellingIsImpossible is the AGENTS.md stock guarantee: 100 buyers race for 10 units and
// exactly 10 orders succeed. The reservation is one conditional UPDATE inside a transaction.
func TestOverSellingIsImpossible(t *testing.T) {
	fixture := newCatalogFixture(t)
	ctx := context.Background()

	var optionID uuid.UUID
	if err := fixture.pool.QueryRow(ctx, `
		SELECT o.id FROM deal_options o
		JOIN deals d ON d.id = o.deal_id
		JOIN merchants m ON m.id = d.merchant_id
		WHERE d.status = 'active' AND d.ends_at > now()
		  AND o.is_active AND m.status = 'active'
		LIMIT 1`).Scan(&optionID); err != nil {
		t.Fatal(err)
	}
	// Leave exactly 10 sellable units, whatever was sold or reserved by the seed.
	if _, err := fixture.pool.Exec(ctx, `
		UPDATE deal_options SET stock_total = stock_sold + stock_reserved + 10 WHERE id = $1`, optionID); err != nil {
		t.Fatal(err)
	}

	userIDs := make([]uuid.UUID, 0, 100)
	rows, err := fixture.pool.Query(ctx, `
		INSERT INTO users (email, name, password_hash)
		SELECT 'buyer' || g || '@test.local', 'Buyer', 'x' FROM generate_series(1, 100) g
		RETURNING id`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		userIDs = append(userIDs, id)
	}
	rows.Close()
	if len(userIDs) != 100 {
		t.Fatalf("created %d buyers, want 100", len(userIDs))
	}

	signer, err := commerce.NewQRSigner("test-qr-secret-with-at-least-32-bytes!!")
	if err != nil {
		t.Fatal(err)
	}
	service, err := commerce.NewService(commerce.Dependencies{
		Repository: store.NewCommerceRepository(fixture.pool),
		Deals:      store.NewCatalogRepository(fixture.pool),
		Signer:     signer,
		Mailer:     mail.NewLogSender(slog.New(slog.NewTextHandler(io.Discard, nil))),
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		Now:        time.Now,
	})
	if err != nil {
		t.Fatal(err)
	}

	var (
		wg        sync.WaitGroup
		succeeded atomic.Int64
		failed    atomic.Int64
		start     = make(chan struct{})
	)
	for _, userID := range userIDs {
		wg.Add(1)
		go func(userID uuid.UUID) {
			defer wg.Done()
			<-start
			_, _, err := service.CreateOrder(ctx, userID, commerce.OrderInput{
				IdempotencyKey: uuid.NewString(),
				Items:          []commerce.OrderItemInput{{OptionID: optionID, Quantity: 1}},
			})
			if err != nil {
				failed.Add(1)
				return
			}
			succeeded.Add(1)
		}(userID)
	}
	close(start)
	wg.Wait()

	if got := succeeded.Load(); got != 10 {
		t.Fatalf("succeeded = %d, want exactly 10 (failed = %d)", got, failed.Load())
	}
	var reserved, sold, total int
	if err := fixture.pool.QueryRow(ctx,
		`SELECT stock_reserved, stock_sold, stock_total FROM deal_options WHERE id = $1`, optionID,
	).Scan(&reserved, &sold, &total); err != nil {
		t.Fatal(err)
	}
	if reserved+sold > total || reserved < 0 || sold < 0 {
		t.Fatalf("stock invariant broken: reserved=%d sold=%d total=%d", reserved, sold, total)
	}
	if reserved != 10 {
		t.Fatalf("reserved = %d, want 10", reserved)
	}
}
