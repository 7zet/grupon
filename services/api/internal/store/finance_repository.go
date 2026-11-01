package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/grupon/grupon/services/api/internal/domain/finance"
	"github.com/grupon/grupon/services/api/internal/domain/ledger"
	sqlc "github.com/grupon/grupon/services/api/internal/store/sqlc"
)

// FinanceRepository performs settlement, payout, refund, dispute, breakage and reconciliation writes.
type FinanceRepository struct {
	pool *pgxpool.Pool
}

// NewFinanceRepository binds the finance writes to a pool.
func NewFinanceRepository(pool *pgxpool.Pool) *FinanceRepository {
	return &FinanceRepository{pool: pool}
}

var _ finance.Repository = (*FinanceRepository)(nil)

// PayableRedemptions lists redemptions ready for payout for one merchant.
func (r *FinanceRepository) PayableRedemptions(ctx context.Context, merchantID uuid.UUID, before time.Time) ([]finance.Payable, error) {
	rows, err := sqlc.New(r.pool).PayableRedemptions(ctx, sqlc.PayableRedemptionsParams{
		MerchantID:     toPgUUID(merchantID),
		EligibleBefore: pgtype.Timestamptz{Time: before, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("list payable redemptions: %w", err)
	}
	items := make([]finance.Payable, 0, len(rows))
	for _, row := range rows {
		items = append(items, finance.Payable{ID: toUUID(row.ID), Amount: row.Amount, Net: row.Amount - row.Commission})
	}
	return items, nil
}

// MerchantOnHold reports whether payouts to the merchant are paused.
func (r *FinanceRepository) MerchantOnHold(ctx context.Context, merchantID uuid.UUID) (bool, error) {
	return sqlc.New(r.pool).MerchantOnHold(ctx, toPgUUID(merchantID))
}

// CreatePayoutRun opens a run awaiting approval and attaches its redemptions, in one transaction.
func (r *FinanceRepository) CreatePayoutRun(ctx context.Context, merchantID uuid.UUID, amount int64, preparedBy uuid.UUID, redemptionIDs []uuid.UUID) (uuid.UUID, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("begin payout run: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	queries := sqlc.New(tx)
	runID, err := queries.InsertPayoutRun(ctx, sqlc.InsertPayoutRunParams{
		MerchantID: toPgUUID(merchantID),
		Amount:     amount,
		PreparedBy: toPgUUID(preparedBy),
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert payout run: %w", err)
	}
	for _, id := range redemptionIDs {
		rows, err := queries.AttachRedemptionToRun(ctx, sqlc.AttachRedemptionToRunParams{RunID: runID, ID: toPgUUID(id)})
		if err != nil {
			return uuid.Nil, fmt.Errorf("attach redemption: %w", err)
		}
		if rows != 1 {
			return uuid.Nil, errors.New("finance: a redemption was already paid out")
		}
	}
	return toUUID(runID), tx.Commit(ctx)
}

// PayoutRun loads a run.
func (r *FinanceRepository) PayoutRun(ctx context.Context, runID uuid.UUID) (finance.PayoutRun, error) {
	row, err := sqlc.New(r.pool).PayoutRunDetail(ctx, toPgUUID(runID))
	if errors.Is(err, pgx.ErrNoRows) {
		return finance.PayoutRun{}, finance.ErrNotFound
	}
	if err != nil {
		return finance.PayoutRun{}, fmt.Errorf("load payout run: %w", err)
	}
	return finance.PayoutRun{
		ID:         toUUID(row.ID),
		MerchantID: toUUID(row.MerchantID),
		Amount:     row.Amount,
		Status:     row.Status,
		PreparedBy: toUUID(row.PreparedBy),
	}, nil
}

// ApprovePayoutRun marks the run sent and posts the payout to the ledger in one transaction.
func (r *FinanceRepository) ApprovePayoutRun(ctx context.Context, run finance.PayoutRun, approverID uuid.UUID, bankDocRef string) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin payout approval: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	queries := sqlc.New(tx)
	locked, err := queries.LockPayoutRun(ctx, toPgUUID(run.ID))
	if err != nil {
		return false, fmt.Errorf("lock payout run: %w", err)
	}
	if locked.Status != "pending_approval" {
		return false, nil
	}
	rows, err := queries.ApprovePayoutRun(ctx, sqlc.ApprovePayoutRunParams{
		ID:         toPgUUID(run.ID),
		ApprovedBy: toPgUUID(approverID),
		BankDocRef: pgtype.Text{String: bankDocRef, Valid: bankDocRef != ""},
	})
	if err != nil {
		return false, fmt.Errorf("approve payout run: %w", err)
	}
	if rows != 1 {
		return false, nil
	}
	if _, err := PostEntry(ctx, queries, ledger.PayoutSent(run.ID.String(), run.MerchantID.String(), run.Amount)); err != nil {
		return false, fmt.Errorf("post payout: %w", err)
	}
	return true, tx.Commit(ctx)
}

// VoucherForRefund loads what a refund decision needs.
func (r *FinanceRepository) VoucherForRefund(ctx context.Context, voucherID uuid.UUID) (finance.VoucherRef, error) {
	row, err := sqlc.New(r.pool).VoucherForRefund(ctx, toPgUUID(voucherID))
	if errors.Is(err, pgx.ErrNoRows) {
		return finance.VoucherRef{}, finance.ErrNotFound
	}
	if err != nil {
		return finance.VoucherRef{}, fmt.Errorf("load voucher: %w", err)
	}
	paidAt := row.CreatedAt.Time
	if row.PaidAt.Valid {
		paidAt = row.PaidAt.Time
	}
	return finance.VoucherRef{
		ID:              toUUID(row.ID),
		BuyerID:         toUUID(row.UserID),
		MerchantID:      toUUID(row.MerchantID),
		Status:          string(row.Status),
		RedemptionsUsed: int(row.RedemptionsUsed),
		UnitPrice:       row.UnitPriceSnapshot,
		PaidAt:          paidAt,
	}, nil
}

// RequestRefund records a refund. An automatic one is approved at once: the voucher is refunded and
// the prepaid value moves to the refund obligation, in one transaction.
func (r *FinanceRepository) RequestRefund(ctx context.Context, voucher finance.VoucherRef, requesterID uuid.UUID, reason string, auto bool) (uuid.UUID, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("begin refund: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	queries := sqlc.New(tx)
	status := "requested"
	if auto {
		status = "approved"
	}
	refundID, err := queries.InsertRefund(ctx, sqlc.InsertRefundParams{
		VoucherID:   toPgUUID(voucher.ID),
		Amount:      voucher.UnitPrice,
		Reason:      reason,
		Status:      status,
		RequestedBy: toPgUUID(requesterID),
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert refund: %w", err)
	}
	if auto {
		rows, err := queries.MarkVoucherRefunded(ctx, toPgUUID(voucher.ID))
		if err != nil {
			return uuid.Nil, fmt.Errorf("refund voucher: %w", err)
		}
		if rows != 1 {
			return uuid.Nil, finance.ErrNotRefundable
		}
		if _, err := PostEntry(ctx, queries, ledger.RefundApproved(toUUID(refundID).String(), voucher.MerchantID.String(), voucher.UnitPrice)); err != nil {
			return uuid.Nil, fmt.Errorf("post refund: %w", err)
		}
	}
	return toUUID(refundID), tx.Commit(ctx)
}

// RefundPaid marks an approved refund as sent and posts it. The approver cannot be the requester.
func (r *FinanceRepository) RefundPaid(ctx context.Context, refundID uuid.UUID, approverID uuid.UUID) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin refund payment: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	queries := sqlc.New(tx)
	refund, err := queries.RefundDetail(ctx, toPgUUID(refundID))
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("load refund: %w", err)
	}
	if refund.Status != "approved" || toUUID(refund.RequestedBy) == approverID {
		return false, finance.ErrForbidden
	}
	rows, err := queries.MarkRefundSucceeded(ctx, sqlc.MarkRefundSucceededParams{
		ID:         toPgUUID(refundID),
		ApproverID: toPgUUID(approverID),
	})
	if err != nil {
		return false, fmt.Errorf("mark refund succeeded: %w", err)
	}
	if rows != 1 {
		return false, nil
	}
	if _, err := PostEntry(ctx, queries, ledger.RefundPaid(refundID.String(), refund.Amount)); err != nil {
		return false, fmt.Errorf("post refund payment: %w", err)
	}
	return true, tx.Commit(ctx)
}

// OpenDispute files a dispute on the buyer's latest redemption. The payout for it is then held.
func (r *FinanceRepository) OpenDispute(ctx context.Context, voucherID, openerID uuid.UUID, reason string) (bool, error) {
	queries := sqlc.New(r.pool)
	latest, err := queries.LatestRedemptionForVoucher(ctx, toPgUUID(voucherID))
	if errors.Is(err, pgx.ErrNoRows) {
		return false, finance.ErrNoRedemption
	}
	if err != nil {
		return false, fmt.Errorf("latest redemption: %w", err)
	}
	disputeID, err := queries.InsertDispute(ctx, sqlc.InsertDisputeParams{
		RedemptionID: latest.ID,
		OpenedBy:     toPgUUID(openerID),
		Reason:       reason,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("insert dispute: %w", err)
	}
	return disputeID.Valid, nil
}

// ExpiredForBreakage lists vouchers whose window closed before the cutoff and that still hold value.
func (r *FinanceRepository) ExpiredForBreakage(ctx context.Context, cutoff time.Time, batch int) ([]finance.BreakageItem, error) {
	rows, err := sqlc.New(r.pool).ExpiredVouchersForBreakage(ctx, sqlc.ExpiredVouchersForBreakageParams{
		Cutoff:    pgtype.Timestamptz{Time: cutoff, Valid: true},
		BatchSize: int32(batch),
	})
	if err != nil {
		return nil, fmt.Errorf("list breakage: %w", err)
	}
	items := make([]finance.BreakageItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, finance.BreakageItem{
			VoucherID:  toUUID(row.ID),
			MerchantID: toUUID(row.MerchantID),
			Remaining:  finance.Remaining(row.UnitPriceSnapshot, int(row.RedemptionsAllowed), int(row.RedemptionsUsed)),
		})
	}
	return items, nil
}

// ApplyBreakage expires one voucher and moves its unused value to the expired account.
func (r *FinanceRepository) ApplyBreakage(ctx context.Context, item finance.BreakageItem) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin breakage: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	queries := sqlc.New(tx)
	rows, err := queries.ExpireVoucherRow(ctx, toPgUUID(item.VoucherID))
	if err != nil {
		return false, fmt.Errorf("expire voucher: %w", err)
	}
	if rows != 1 {
		return false, nil
	}
	if item.Remaining > 0 {
		if _, err := PostEntry(ctx, queries, ledger.Expired(item.VoucherID.String(), item.MerchantID.String(), item.Remaining)); err != nil {
			return false, fmt.Errorf("post breakage: %w", err)
		}
	}
	return true, tx.Commit(ctx)
}

// ActiveVouchers lists the unused value of every active voucher.
func (r *FinanceRepository) ActiveVouchers(ctx context.Context) ([]finance.VoucherValue, error) {
	rows, err := sqlc.New(r.pool).ActiveVouchersForValue(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active vouchers: %w", err)
	}
	values := make([]finance.VoucherValue, 0, len(rows))
	for _, row := range rows {
		values = append(values, finance.VoucherValue{
			MerchantID: toUUID(row.MerchantID),
			Remaining:  finance.Remaining(row.UnitPriceSnapshot, int(row.RedemptionsAllowed), int(row.RedemptionsUsed)),
		})
	}
	return values, nil
}

// CustomerPrepaidBalance is the ledger's total of buyers' prepaid value.
func (r *FinanceRepository) CustomerPrepaidBalance(ctx context.Context) (int64, error) {
	return sqlc.New(r.pool).CustomerPrepaidTotal(ctx)
}

// RecordFinding stores a reconciliation mismatch for finance to review.
func (r *FinanceRepository) RecordFinding(ctx context.Context, check string, expected, actual int64) error {
	return sqlc.New(r.pool).InsertReconciliationFinding(ctx, sqlc.InsertReconciliationFindingParams{
		CheckName: check,
		Expected:  expected,
		Actual:    actual,
	})
}

// MarkPayoutReconciled moves a sent payout run to reconciled after the bank statement matched it.
func (r *FinanceRepository) MarkPayoutReconciled(ctx context.Context, runID uuid.UUID) (bool, error) {
	rows, err := sqlc.New(r.pool).MarkPayoutRunReconciled(ctx, toPgUUID(runID))
	if err != nil {
		return false, fmt.Errorf("reconcile payout: %w", err)
	}
	return rows == 1, nil
}

// ApproveRefund approves a requested refund and posts the ledger entry in one transaction.
func (r *FinanceRepository) ApproveRefund(ctx context.Context, refundID, approverID uuid.UUID) (finance.RefundRef, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return finance.RefundRef{}, fmt.Errorf("begin refund approval: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	queries := sqlc.New(tx)
	row, err := queries.ApproveRequestedRefund(ctx, sqlc.ApproveRequestedRefundParams{
		ID:         toPgUUID(refundID),
		ApproverID: toPgUUID(approverID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return finance.RefundRef{}, finance.ErrNotPending
	}
	if err != nil {
		return finance.RefundRef{}, fmt.Errorf("approve refund: %w", err)
	}
	ref := finance.RefundRef{ID: toUUID(row.ID), MerchantID: toUUID(row.MerchantID), Amount: row.Amount}
	// An unused voucher is taken out of the buyer's prepaid value and closed. A redeemed one (dispute)
	// is paid back by the merchant, whose payable already holds the redeemed amount.
	rows, err := queries.MarkVoucherRefunded(ctx, row.VoucherID)
	if err != nil {
		return finance.RefundRef{}, fmt.Errorf("refund voucher: %w", err)
	}
	entry := ledger.RefundFromMerchant(ref.ID.String(), ref.MerchantID.String(), ref.Amount)
	if rows == 1 {
		entry = ledger.RefundApproved(ref.ID.String(), ref.MerchantID.String(), ref.Amount)
	}
	if _, err := PostEntry(ctx, queries, entry); err != nil {
		return finance.RefundRef{}, fmt.Errorf("post refund approval: %w", err)
	}
	return ref, tx.Commit(ctx)
}

// ResolveDispute closes an open dispute. A customer resolution also files a refund request for the
// redemption amount; the resolver cannot approve it, so a second person must.
func (r *FinanceRepository) ResolveDispute(ctx context.Context, disputeID, resolverID uuid.UUID, resolution string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin dispute resolution: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	queries := sqlc.New(tx)
	dispute, err := queries.DisputeForResolution(ctx, toPgUUID(disputeID))
	if errors.Is(err, pgx.ErrNoRows) {
		return finance.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("load dispute: %w", err)
	}
	if dispute.Status != "open" {
		return finance.ErrDisputeClosed
	}
	status := "resolved_merchant"
	if resolution == "customer" {
		status = "resolved_customer"
	}
	rows, err := queries.ResolveDispute(ctx, sqlc.ResolveDisputeParams{ID: toPgUUID(disputeID), Status: status})
	if err != nil {
		return fmt.Errorf("close dispute: %w", err)
	}
	if rows != 1 {
		return finance.ErrDisputeClosed
	}
	if resolution == "customer" {
		if _, err := queries.InsertRefund(ctx, sqlc.InsertRefundParams{
			VoucherID:   dispute.VoucherID,
			Amount:      dispute.Amount,
			Reason:      "dispute resolved for customer",
			Status:      "requested",
			RequestedBy: toPgUUID(resolverID),
		}); err != nil {
			return fmt.Errorf("file dispute refund: %w", err)
		}
	}
	return tx.Commit(ctx)
}
