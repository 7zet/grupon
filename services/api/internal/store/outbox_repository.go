package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/grupon/grupon/services/api/internal/domain/outbox"
	sqlc "github.com/grupon/grupon/services/api/internal/store/sqlc"
)

// OutboxRepository implements outbox.Store on PostgreSQL.
type OutboxRepository struct {
	pool *pgxpool.Pool
}

// NewOutboxRepository returns the outbox store.
func NewOutboxRepository(pool *pgxpool.Pool) *OutboxRepository {
	return &OutboxRepository{pool: pool}
}

// PublishPending locks up to batch unpublished rows (SKIP LOCKED lets several workers share the work),
// publishes them and marks them in the same transaction. A publish error rolls the whole batch back.
func (r *OutboxRepository) PublishPending(ctx context.Context, batch int, publish outbox.Publisher) (int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin outbox batch: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	queries := sqlc.New(tx)
	rows, err := queries.ClaimOutboxBatch(ctx, int32(batch))
	if err != nil {
		return 0, fmt.Errorf("claim outbox: %w", err)
	}
	for _, row := range rows {
		if err := publish(ctx, row.Topic, row.Payload); err != nil {
			return 0, fmt.Errorf("publish %s: %w", row.Topic, err)
		}
		if err := queries.MarkOutboxPublished(ctx, row.ID); err != nil {
			return 0, fmt.Errorf("mark outbox row: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit outbox batch: %w", err)
	}
	return len(rows), nil
}
