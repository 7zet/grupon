package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/grupon/grupon/services/api/internal/domain/commerce"
	"github.com/grupon/grupon/services/api/internal/domain/redemption"
	sqlc "github.com/grupon/grupon/services/api/internal/store/sqlc"
)

// RedemptionRepository reads vouchers for staff and performs the conditional redemption write.
type RedemptionRepository struct {
	pool     *pgxpool.Pool
	commerce *CommerceRepository
	catalog  *CatalogRepository
}

// NewRedemptionRepository binds voucher reads and the redemption write to the pool.
func NewRedemptionRepository(pool *pgxpool.Pool, commerce *CommerceRepository, catalog *CatalogRepository) *RedemptionRepository {
	return &RedemptionRepository{pool: pool, commerce: commerce, catalog: catalog}
}

var _ redemption.Repository = (*RedemptionRepository)(nil)

// Find loads a voucher by id.
func (r *RedemptionRepository) Find(ctx context.Context, voucherID uuid.UUID) (commerce.Voucher, error) {
	row, err := sqlc.New(r.pool).GetVoucherByID(ctx, toPgUUID(voucherID))
	if errors.Is(err, pgx.ErrNoRows) {
		return commerce.Voucher{}, redemption.ErrNotFound
	}
	if err != nil {
		return commerce.Voucher{}, fmt.Errorf("get voucher: %w", err)
	}
	return voucherFromFields(row.ID, row.OrderItemID, row.UserID, row.DealID, row.DealOptionID,
		row.MerchantID, row.Code, row.QrToken, string(row.Status), row.ValidFrom, row.ValidUntil, row.RedeemedAt), nil
}

// FindByCode loads a voucher by its human code.
func (r *RedemptionRepository) FindByCode(ctx context.Context, code string) (commerce.Voucher, error) {
	row, err := sqlc.New(r.pool).GetVoucherByCode(ctx, code)
	if errors.Is(err, pgx.ErrNoRows) {
		return commerce.Voucher{}, redemption.ErrNotFound
	}
	if err != nil {
		return commerce.Voucher{}, fmt.Errorf("get voucher by code: %w", err)
	}
	return voucherFromFields(row.ID, row.OrderItemID, row.UserID, row.DealID, row.DealOptionID,
		row.MerchantID, row.Code, row.QrToken, string(row.Status), row.ValidFrom, row.ValidUntil, row.RedeemedAt), nil
}

// LocationOwned reports whether a branch belongs to the merchant.
func (r *RedemptionRepository) LocationOwned(ctx context.Context, merchantID, locationID uuid.UUID) (bool, error) {
	count, err := sqlc.New(r.pool).CountOwnedLocations(ctx, sqlc.CountOwnedLocationsParams{
		MerchantID:  toPgUUID(merchantID),
		LocationIds: toPgUUIDs([]uuid.UUID{locationID}),
	})
	if err != nil {
		return false, fmt.Errorf("check location: %w", err)
	}
	return count == 1, nil
}

// Redeem updates the voucher only if it is still active and in its window, then writes the audit row.
func (r *RedemptionRepository) Redeem(ctx context.Context, merchantID, voucherID, locationID uuid.UUID, at time.Time, audit redemption.AuditEntry) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin redemption: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	queries := sqlc.New(tx)
	rows, err := queries.RedeemVoucherIfActive(ctx, sqlc.RedeemVoucherIfActiveParams{
		RedeemedAt:       pgTime(at),
		RedeemedByUserID: toPgUUID(audit.ActorID),
		LocationID:       toPgUUID(locationID),
		ID:               toPgUUID(voucherID),
		MerchantID:       toPgUUID(merchantID),
	})
	if err != nil {
		return false, fmt.Errorf("redeem voucher: %w", err)
	}
	if rows == 0 {
		return false, nil
	}
	after, err := json.Marshal(audit.After)
	if err != nil {
		return false, fmt.Errorf("encode audit: %w", err)
	}
	if err := queries.InsertAuditLog(ctx, sqlc.InsertAuditLogParams{
		ActorUserID: toPgUUID(audit.ActorID),
		ActorRole:   sqlc.AppRole(audit.Role),
		Action:      audit.Action,
		EntityType:  "voucher",
		EntityID:    toPgUUID(audit.VoucherID),
		AfterData:   after,
	}); err != nil {
		return false, fmt.Errorf("write audit log: %w", err)
	}
	return true, tx.Commit(ctx)
}

// View attaches the option and deal to a voucher for display.
func (r *RedemptionRepository) View(ctx context.Context, voucher commerce.Voucher) (commerce.VoucherView, error) {
	records, err := r.commerce.OptionRecords(ctx, []uuid.UUID{voucher.OptionID})
	if err != nil {
		return commerce.VoucherView{}, fmt.Errorf("load voucher option: %w", err)
	}
	summaries, err := r.catalog.DealSummaries(ctx, []uuid.UUID{voucher.DealID})
	if err != nil {
		return commerce.VoucherView{}, fmt.Errorf("load voucher deal: %w", err)
	}
	if len(records) != 1 || len(summaries) != 1 {
		return commerce.VoucherView{}, redemption.ErrNotFound
	}
	return commerce.VoucherView{Voucher: voucher, Option: records[0], Deal: summaries[0]}, nil
}
