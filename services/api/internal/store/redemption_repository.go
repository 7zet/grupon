package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/grupon/grupon/services/api/internal/domain/catalog"
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

// FindByBackupCode loads a voucher by the stored HMAC of its backup code.
func (r *RedemptionRepository) FindByBackupCode(ctx context.Context, codeHMAC string) (commerce.Voucher, error) {
	id, err := sqlc.New(r.pool).VoucherIDByBackupCodeHMAC(ctx, pgtype.Text{String: codeHMAC, Valid: true})
	if errors.Is(err, pgx.ErrNoRows) {
		return commerce.Voucher{}, redemption.ErrNotFound
	}
	if err != nil {
		return commerce.Voucher{}, fmt.Errorf("get voucher by backup code: %w", err)
	}
	return r.Find(ctx, toUUID(id))
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

// Redeem records one use of the voucher; see UseVoucher for the transactional rules.
func (r *RedemptionRepository) Redeem(ctx context.Context, merchantID, voucherID, locationID uuid.UUID, at time.Time, audit redemption.AuditEntry, geo *redemption.Geo) (bool, error) {
	return r.UseVoucher(ctx, merchantID, voucherID, locationID, at, audit, geo)
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

func (r *RedemptionRepository) History(ctx context.Context, merchantID uuid.UUID, staffID *uuid.UUID, page catalog.Page) ([]redemption.HistoryItem, int64, error) {
	q := sqlc.New(r.pool)
	staff := pgtype.UUID{}
	if staffID != nil {
		staff = toPgUUID(*staffID)
	}
	params := sqlc.ListMerchantRedemptionHistoryParams{MerchantID: toPgUUID(merchantID), StaffUserID: staff, PageOffset: int32(page.Offset()), PageLimit: int32(page.Size)}
	rows, err := q.ListMerchantRedemptionHistory(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("list redemption history: %w", err)
	}
	total, err := q.CountMerchantRedemptionHistory(ctx, sqlc.CountMerchantRedemptionHistoryParams{MerchantID: toPgUUID(merchantID), StaffUserID: staff})
	if err != nil {
		return nil, 0, fmt.Errorf("count redemption history: %w", err)
	}
	items := make([]redemption.HistoryItem, 0, len(rows))
	for _, row := range rows {
		deal, err := catalog.ParseText(row.DealTitle)
		if err != nil {
			return nil, 0, err
		}
		option, err := catalog.ParseText(row.OptionTitle)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, redemption.HistoryItem{ID: toUUID(row.ID), Time: row.CreatedAt.Time, Deal: deal, Option: option, Buyer: row.BuyerName, Branch: row.Branch, Cashier: row.CashierName, Reversed: row.Reversed})
	}
	return items, total, nil
}

// AttemptStats counts a staff member's scanner lookups for the limits.
func (r *RedemptionRepository) AttemptStats(ctx context.Context, staffID uuid.UUID, minuteSince, blockSince time.Time) (redemption.AttemptCounts, error) {
	row, err := sqlc.New(r.pool).RedemptionAttemptStats(ctx, sqlc.RedemptionAttemptStatsParams{
		StaffUserID: toPgUUID(staffID),
		MinuteSince: pgtype.Timestamptz{Time: minuteSince, Valid: true},
		BlockSince:  pgtype.Timestamptz{Time: blockSince, Valid: true},
	})
	if err != nil {
		return redemption.AttemptCounts{}, fmt.Errorf("attempt stats: %w", err)
	}
	return redemption.AttemptCounts{LastMinute: int(row.LastMinute), FailedRecent: int(row.FailedRecent)}, nil
}

// RecordAttempt stores one scanner lookup and whether it failed.
// RecordDenied writes an audit row for a refused scan. The voucher is untouched.
func (r *RedemptionRepository) RecordDenied(ctx context.Context, audit redemption.AuditEntry) error {
	payload, err := json.Marshal(audit.After)
	if err != nil {
		return fmt.Errorf("encode audit: %w", err)
	}
	if err := sqlc.New(r.pool).InsertAuditLog(ctx, sqlc.InsertAuditLogParams{
		ActorUserID: toPgUUID(audit.ActorID),
		ActorRole:   sqlc.AppRole(audit.Role),
		Action:      audit.Action,
		EntityType:  "voucher",
		EntityID:    toPgUUID(audit.VoucherID),
		AfterData:   payload,
	}); err != nil {
		return fmt.Errorf("write audit log: %w", err)
	}
	return nil
}

func (r *RedemptionRepository) RecordAttempt(ctx context.Context, staffID uuid.UUID, failed bool) error {
	if err := sqlc.New(r.pool).InsertRedemptionAttempt(ctx, sqlc.InsertRedemptionAttemptParams{
		StaffUserID: toPgUUID(staffID),
		Failed:      failed,
	}); err != nil {
		return fmt.Errorf("record attempt: %w", err)
	}
	return nil
}
