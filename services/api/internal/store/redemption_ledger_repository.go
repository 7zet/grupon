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

	"github.com/grupon/grupon/services/api/internal/domain/ledger"
	"github.com/grupon/grupon/services/api/internal/domain/redemption"
	sqlc "github.com/grupon/grupon/services/api/internal/store/sqlc"
)

// fraudBurstLimit is how many redemptions one staff member may make in a minute before a signal is raised.
const fraudBurstLimit = 5

// UseVoucher records one use of a voucher inside a single transaction: the row is locked, the count
// and status change, the redemption row and its ledger entry are written, and the audit row follows.
// It returns false when the voucher cannot be used at this moment (BR-03, BR-08).
func (r *RedemptionRepository) UseVoucher(ctx context.Context, merchantID, voucherID, locationID uuid.UUID, at time.Time, audit redemption.AuditEntry) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin redemption: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	queries := sqlc.New(tx)
	voucher, err := queries.LockVoucherForRedemption(ctx, sqlc.LockVoucherForRedemptionParams{
		ID:         toPgUUID(voucherID),
		MerchantID: toPgUUID(merchantID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("lock voucher: %w", err)
	}
	if voucher.Status != sqlc.VoucherStatusActive || voucher.RedemptionsUsed >= voucher.RedemptionsAllowed {
		return false, nil
	}
	if !at.Before(voucher.ValidUntil.Time) || at.Before(voucher.ValidFrom.Time) {
		return false, nil
	}

	sequence := int(voucher.RedemptionsUsed) + 1
	shares, err := ledger.Split(voucher.UnitPriceSnapshot, int(voucher.RedemptionsAllowed))
	if err != nil {
		return false, fmt.Errorf("split voucher value: %w", err)
	}
	amount := shares[sequence-1]
	commission := ledger.Commission(amount, int(voucher.CommissionBps))

	rows, err := queries.ApplyVoucherUse(ctx, sqlc.ApplyVoucherUseParams{
		ID:          toPgUUID(voucherID),
		StaffUserID: toPgUUID(audit.ActorID),
		LocationID:  toPgUUID(locationID),
	})
	if err != nil {
		return false, fmt.Errorf("apply voucher use: %w", err)
	}
	if rows != 1 {
		return false, nil
	}
	redemptionID, err := queries.InsertRedemption(ctx, sqlc.InsertRedemptionParams{
		VoucherID:   toPgUUID(voucherID),
		MerchantID:  toPgUUID(merchantID),
		LocationID:  toPgUUID(locationID),
		StaffUserID: toPgUUID(audit.ActorID),
		Sequence:    int32(sequence),
		Amount:      amount,
		Commission:  commission,
	})
	if err != nil {
		return false, fmt.Errorf("insert redemption: %w", err)
	}
	// Fraud rule (PRD §12.1): more than five redemptions by one staff member in a minute is a burst. The
	// merchant's payouts are held until a fraud analyst reviews it.
	recent, err := queries.RecentStaffRedemptions(ctx, sqlc.RecentStaffRedemptionsParams{
		StaffUserID: toPgUUID(audit.ActorID),
		Since:       pgtype.Timestamptz{Time: at.Add(-time.Minute), Valid: true},
	})
	if err != nil {
		return false, fmt.Errorf("count recent redemptions: %w", err)
	}
	if recent > fraudBurstLimit {
		detail, _ := json.Marshal(map[string]any{"count": recent, "window": "1m"})
		if err := queries.InsertFraudSignal(ctx, sqlc.InsertFraudSignalParams{
			MerchantID:    toPgUUID(merchantID),
			SubjectUserID: toPgUUID(audit.ActorID),
			Rule:          "redemption_burst",
			Detail:        detail,
		}); err != nil {
			return false, fmt.Errorf("record fraud signal: %w", err)
		}
		if err := queries.PlaceMerchantHold(ctx, sqlc.PlaceMerchantHoldParams{
			MerchantID: toPgUUID(merchantID),
			Reason:     "redemption burst under review",
		}); err != nil {
			return false, fmt.Errorf("hold merchant payouts: %w", err)
		}
	}
	if _, err := PostEntry(ctx, queries, ledger.Redeemed(toUUID(redemptionID).String(), merchantID.String(), amount, commission)); err != nil {
		return false, fmt.Errorf("post redemption: %w", err)
	}
	if err := writeVoucherAudit(ctx, queries, audit, voucherID, toUUID(redemptionID)); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

// ReversalCandidate returns the latest open redemption of a voucher for the merchant.
func (r *RedemptionRepository) ReversalCandidate(ctx context.Context, merchantID, voucherID uuid.UUID) (redemption.Redemption, error) {
	row, err := sqlc.New(r.pool).LatestOpenRedemption(ctx, sqlc.LatestOpenRedemptionParams{
		VoucherID:  toPgUUID(voucherID),
		MerchantID: toPgUUID(merchantID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return redemption.Redemption{}, redemption.ErrNotFound
	}
	if err != nil {
		return redemption.Redemption{}, fmt.Errorf("latest redemption: %w", err)
	}
	return redemption.Redemption{
		ID:         toUUID(row.ID),
		VoucherID:  toUUID(row.VoucherID),
		StaffID:    toUUID(row.StaffUserID),
		Amount:     row.Amount,
		Commission: row.Commission,
		CreatedAt:  row.CreatedAt.Time,
	}, nil
}

// ReverseUse undoes a redemption: the row is marked reversed, the voucher count goes back, and a
// reversing ledger entry is written. Nothing is edited in place (ADR-0005).
func (r *RedemptionRepository) ReverseUse(ctx context.Context, merchantID uuid.UUID, found redemption.Redemption, reason string, audit redemption.AuditEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin reversal: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	queries := sqlc.New(tx)
	if _, err := queries.LatestOpenRedemption(ctx, sqlc.LatestOpenRedemptionParams{
		VoucherID:  toPgUUID(found.VoucherID),
		MerchantID: toPgUUID(merchantID),
	}); err != nil {
		return redemption.ErrNotFound
	}
	rows, err := queries.MarkRedemptionReversed(ctx, sqlc.MarkRedemptionReversedParams{
		ID:          toPgUUID(found.ID),
		ActorUserID: toPgUUID(audit.ActorID),
		Reason:      pgtype.Text{String: reason, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("mark reversed: %w", err)
	}
	if rows != 1 {
		return redemption.ErrNotFound
	}
	if rows, err := queries.RollBackVoucherUse(ctx, toPgUUID(found.VoucherID)); err != nil || rows != 1 {
		if err == nil {
			err = errors.New("voucher count was already zero")
		}
		return fmt.Errorf("roll back voucher: %w", err)
	}
	if _, err := PostEntry(ctx, queries, ledger.Reversed(found.ID.String(), merchantID.String(), found.Amount, found.Commission)); err != nil {
		return fmt.Errorf("post reversal: %w", err)
	}
	if err := writeVoucherAudit(ctx, queries, audit, found.VoucherID, found.ID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func writeVoucherAudit(ctx context.Context, queries *sqlc.Queries, audit redemption.AuditEntry, voucherID, redemptionID uuid.UUID) error {
	after := map[string]any{"redemption_id": redemptionID.String()}
	for key, value := range audit.After {
		after[key] = value
	}
	payload, err := json.Marshal(after)
	if err != nil {
		return fmt.Errorf("encode audit: %w", err)
	}
	if err := queries.InsertAuditLog(ctx, sqlc.InsertAuditLogParams{
		ActorUserID: toPgUUID(audit.ActorID),
		ActorRole:   sqlc.AppRole(audit.Role),
		Action:      audit.Action,
		EntityType:  "voucher",
		EntityID:    toPgUUID(voucherID),
		AfterData:   payload,
	}); err != nil {
		return fmt.Errorf("write audit log: %w", err)
	}
	return nil
}
