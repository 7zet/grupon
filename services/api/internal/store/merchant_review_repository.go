package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dealo/dealo/services/api/internal/domain/merchant"
	sqlc "github.com/dealo/dealo/services/api/internal/store/sqlc"
)

// MerchantReviewRepository implements merchant.ReviewRepository on PostgreSQL.
type MerchantReviewRepository struct {
	pool *pgxpool.Pool
}

// NewMerchantReviewRepository returns the review store.
func NewMerchantReviewRepository(pool *pgxpool.Pool) *MerchantReviewRepository {
	return &MerchantReviewRepository{pool: pool}
}

// ListForReview returns the queue, oldest submission first.
func (r *MerchantReviewRepository) ListForReview(ctx context.Context, status string) ([]merchant.ReviewItem, error) {
	rows, err := sqlc.New(r.pool).ListMerchantsForReview(ctx, optText(status))
	if err != nil {
		return nil, fmt.Errorf("list businesses for review: %w", err)
	}
	items := make([]merchant.ReviewItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, merchant.ReviewItem{
			ID:              toUUID(row.ID),
			LegalName:       row.LegalName,
			DisplayName:     row.DisplayName,
			STIR:            row.Stir.String,
			OrgForm:         row.OrgForm.String,
			TaxRegime:       row.TaxRegime.String,
			BankMFO:         row.BankMfo.String,
			BankAccount:     row.BankAccount.String,
			ResponsibleName: row.ResponsibleName.String,
			ReviewStatus:    row.ReviewStatus,
			SubmittedAt:     row.SubmittedAt.Time,
			Documents:       row.Documents,
		})
	}
	return items, nil
}

// Transition moves the business and writes its audit row in one transaction. The status update is
// conditional, so two reviewers cannot both decide the same case.
func (r *MerchantReviewRepository) Transition(ctx context.Context, merchantID uuid.UUID, from []string, to, reason string, audit merchant.ReviewAudit) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin business review: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	queries := sqlc.New(tx)
	rows, err := queries.TransitionMerchantReview(ctx, sqlc.TransitionMerchantReviewParams{
		ToStatus:     to,
		Reason:       pgtype.Text{String: reason, Valid: reason != ""},
		ID:           toPgUUID(merchantID),
		FromStatuses: from,
	})
	if err != nil {
		return fmt.Errorf("transition business: %w", err)
	}
	if rows == 0 {
		if _, err := queries.MerchantReviewStatusByID(ctx, toPgUUID(merchantID)); errors.Is(err, pgx.ErrNoRows) {
			return merchant.ErrBusinessNotFound
		}
		return merchant.ErrInvalidTransition
	}
	after, err := json.Marshal(audit.After)
	if err != nil {
		return fmt.Errorf("encode audit: %w", err)
	}
	if err := queries.InsertAuditLog(ctx, sqlc.InsertAuditLogParams{
		ActorUserID: toPgUUID(audit.ActorID),
		ActorRole:   sqlc.AppRole(audit.Role),
		Action:      audit.Action,
		EntityType:  "merchant",
		EntityID:    toPgUUID(merchantID),
		AfterData:   after,
	}); err != nil {
		return fmt.Errorf("write audit: %w", err)
	}
	return tx.Commit(ctx)
}

// ReviewStatus returns the review status of a business, or ErrBusinessNotFound.
func (r *MerchantReviewRepository) ReviewStatus(ctx context.Context, merchantID uuid.UUID) (string, error) {
	status, err := sqlc.New(r.pool).MerchantReviewStatusByID(ctx, toPgUUID(merchantID))
	if errors.Is(err, pgx.ErrNoRows) {
		return "", merchant.ErrBusinessNotFound
	}
	if err != nil {
		return "", fmt.Errorf("load review status: %w", err)
	}
	return status, nil
}
