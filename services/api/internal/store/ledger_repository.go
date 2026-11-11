package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/dealo/dealo/services/api/internal/domain/ledger"
	sqlc "github.com/dealo/dealo/services/api/internal/store/sqlc"
)

// PostEntry writes a balanced entry inside the caller's transaction. It returns false when an entry
// with the same idempotency key already exists, so a retried event never posts twice.
func PostEntry(ctx context.Context, q *sqlc.Queries, entry ledger.Entry) (bool, error) {
	if err := entry.Validate(); err != nil {
		return false, err
	}
	exists, err := q.LedgerEntryExists(ctx, entry.IdempotencyKey)
	if err != nil {
		return false, fmt.Errorf("check ledger entry: %w", err)
	}
	if exists {
		return false, nil
	}
	entryID, err := q.InsertJournalEntry(ctx, sqlc.InsertJournalEntryParams{
		Kind:           entry.Kind,
		Reference:      entry.Reference,
		IdempotencyKey: entry.IdempotencyKey,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return false, nil
		}
		return false, fmt.Errorf("insert journal entry: %w", err)
	}
	for _, line := range entry.Lines {
		accountID, err := q.EnsureLedgerAccount(ctx, sqlc.EnsureLedgerAccountParams{
			Code: line.Account,
			Kind: accountKind(line.Account),
		})
		if err != nil {
			return false, fmt.Errorf("ledger account %s: %w", line.Account, err)
		}
		if err := q.InsertJournalLine(ctx, sqlc.InsertJournalLineParams{
			EntryID:   entryID,
			AccountID: accountID,
			Direction: string(line.Direction),
			Amount:    line.Amount,
		}); err != nil {
			return false, fmt.Errorf("insert journal line: %w", err)
		}
	}
	payload, err := json.Marshal(map[string]string{"kind": entry.Kind, "reference": entry.Reference})
	if err != nil {
		return false, fmt.Errorf("encode outbox payload: %w", err)
	}
	if err := q.InsertOutbox(ctx, sqlc.InsertOutboxParams{Topic: "ledger.posted", Payload: payload}); err != nil {
		return false, fmt.Errorf("write outbox: %w", err)
	}
	return true, nil
}

// accountKind classifies an account code. Account kinds drive the balance sign in reports.
func accountKind(code string) string {
	switch {
	case strings.HasPrefix(code, ledger.CustomerPrepaidExp),
		strings.HasPrefix(code, ledger.CustomerPrepaid),
		strings.HasPrefix(code, ledger.MerchantPayable),
		code == ledger.RefundsPayable:
		return "liability"
	case code == ledger.CommissionRevenue, code == ledger.BreakageRevenue:
		return "revenue"
	case code == ledger.PSPFees:
		return "expense"
	default:
		return "asset"
	}
}
