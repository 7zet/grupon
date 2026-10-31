package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/grupon/grupon/services/api/internal/domain/catalog"
	"github.com/grupon/grupon/services/api/internal/domain/merchant"
	"github.com/grupon/grupon/services/api/internal/domain/moderation"
	sqlc "github.com/grupon/grupon/services/api/internal/store/sqlc"
)

// ModerationRepository persists review decisions. Reads of a deal reuse the merchant deal read model.
type ModerationRepository struct {
	pool    *pgxpool.Pool
	catalog *CatalogRepository
	deals   *MerchantDealRepository
}

// NewModerationRepository binds review decisions to the pool and the shared deal read models.
func NewModerationRepository(pool *pgxpool.Pool, catalog *CatalogRepository, deals *MerchantDealRepository) *ModerationRepository {
	return &ModerationRepository{pool: pool, catalog: catalog, deals: deals}
}

var _ moderation.Repository = (*ModerationRepository)(nil)

// ListPendingIDs returns one page of deals waiting for review and the total count.
func (r *ModerationRepository) ListPendingIDs(ctx context.Context, page catalog.Page) ([]uuid.UUID, int64, error) {
	queries := sqlc.New(r.pool)
	total, err := queries.CountPendingDeals(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count pending deals: %w", err)
	}
	ids, err := queries.ListPendingDealIDs(ctx, sqlc.ListPendingDealIDsParams{
		PageLimit:  int32(page.Size),
		PageOffset: int32(page.Offset()),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list pending deals: %w", err)
	}
	return toUUIDs(ids), total, nil
}

// Summaries returns card views for the given deals, in the order requested.
func (r *ModerationRepository) Summaries(ctx context.Context, ids []uuid.UUID) ([]catalog.DealSummary, error) {
	return r.catalog.DealSummaries(ctx, ids)
}

// Record reads the review state of a deal.
func (r *ModerationRepository) Record(ctx context.Context, dealID uuid.UUID) (moderation.Record, error) {
	row, err := sqlc.New(r.pool).GetDealReviewRow(ctx, toPgUUID(dealID))
	if errors.Is(err, pgx.ErrNoRows) {
		return moderation.Record{}, moderation.ErrNotFound
	}
	if err != nil {
		return moderation.Record{}, fmt.Errorf("get deal review row: %w", err)
	}
	return moderation.Record{
		ID:         toUUID(row.ID),
		MerchantID: toUUID(row.MerchantID),
		Status:     string(row.Status),
		StartsAt:   row.StartsAt.Time,
	}, nil
}

// Transition moves a deal out of review only if it is still pending, and writes the audit row in the same transaction.
func (r *ModerationRepository) Transition(ctx context.Context, dealID uuid.UUID, to, reason string, audit moderation.AuditEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin moderation: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	queries := sqlc.New(tx)
	rejection := pgtype.Text{String: reason, Valid: reason != ""}
	rows, err := queries.TransitionPendingDeal(ctx, sqlc.TransitionPendingDealParams{
		Status:          sqlc.DealStatus(to),
		RejectionReason: rejection,
		ID:              toPgUUID(dealID),
	})
	if err != nil {
		return fmt.Errorf("transition deal: %w", err)
	}
	if rows == 0 {
		return moderation.ErrNotPending
	}
	if err := writeAudit(ctx, queries, merchant.AuditEntry{
		ActorID: audit.ActorID,
		Role:    audit.Role,
		Action:  audit.Action,
		DealID:  dealID,
		After:   audit.After,
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Detail returns the full deal for review, regardless of its workflow status.
func (r *ModerationRepository) Detail(ctx context.Context, dealID uuid.UUID) (catalog.Deal, error) {
	record, err := r.Record(ctx, dealID)
	if err != nil {
		return catalog.Deal{}, err
	}
	deal, err := r.deals.Detail(ctx, record.MerchantID, dealID)
	if errors.Is(err, merchant.ErrDealNotFound) {
		return catalog.Deal{}, moderation.ErrNotFound
	}
	return deal, err
}
