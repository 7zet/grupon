// Package finance runs settlement, payouts with four-eyes approval, refunds, disputes, breakage and
// reconciliation (Dealo PRD §6 FR-06…08, §8, §12). Every money movement is a balanced ledger entry.
package finance

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/grupon/grupon/services/api/internal/domain/ledger"
)

// Windows from the PRD. Amounts are tiyin.
const (
	PayoutDisputeWindow = 3 * 24 * time.Hour  // FR-07.2: redemptions wait 3 days before payout
	RefundWindow        = 14 * 24 * time.Hour // FR-06: unused voucher refunded automatically
	BreakageWindow      = 90 * 24 * time.Hour // PRD §8.5: 90 days after the voucher window closed
)

// Errors for the finance rules. The HTTP layer maps them to problem codes.
var (
	ErrForbidden      = errors.New("finance: role may not do this")
	ErrNothingToPay   = errors.New("finance: no redemptions are ready for payout")
	ErrMerchantOnHold = errors.New("finance: merchant is on payout hold")
	ErrNotPending     = errors.New("finance: payout is not waiting for approval")
	ErrSelfApproval   = errors.New("finance: the person who prepared a payout cannot approve it")
	ErrNotFound       = errors.New("finance: record not found")
	ErrNotOwner       = errors.New("finance: voucher belongs to another buyer")
	ErrNotRefundable  = errors.New("finance: voucher cannot be refunded")
	ErrDisputeExists  = errors.New("finance: a dispute is already open for this redemption")
	ErrNoRedemption   = errors.New("finance: voucher has no redemption to dispute")
)

// Actor is the staff or buyer making the request.
type Actor struct {
	UserID uuid.UUID
	Role   string
}

// Payable is one redemption ready to be paid out.
type Payable struct {
	ID     uuid.UUID
	Amount int64
	// Net is what the merchant keeps: the redeemed amount minus the platform commission.
	Net int64
}

// PayoutRun is a batch of payable redemptions for one merchant.
type PayoutRun struct {
	ID         uuid.UUID
	MerchantID uuid.UUID
	Amount     int64
	Status     string
	PreparedBy uuid.UUID
}

// VoucherRef is what a refund or dispute needs to know about a voucher.
type VoucherRef struct {
	ID              uuid.UUID
	BuyerID         uuid.UUID
	MerchantID      uuid.UUID
	Status          string
	RedemptionsUsed int
	UnitPrice       int64
	PaidAt          time.Time
}

// BreakageItem is a voucher whose window closed long enough ago to be recognised as breakage.
type BreakageItem struct {
	VoucherID  uuid.UUID
	MerchantID uuid.UUID
	Remaining  int64
}

// VoucherValue is the unused value a voucher still represents, for reconciliation.
type VoucherValue struct {
	MerchantID uuid.UUID
	Remaining  int64
}

// Repository performs the writes. Each method is one transaction that also writes its ledger entry
// and outbox event, so a money movement is either fully recorded or not at all.
type Repository interface {
	PayableRedemptions(ctx context.Context, merchantID uuid.UUID, before time.Time) ([]Payable, error)
	MerchantOnHold(ctx context.Context, merchantID uuid.UUID) (bool, error)
	CreatePayoutRun(ctx context.Context, merchantID uuid.UUID, amount int64, preparedBy uuid.UUID, redemptionIDs []uuid.UUID) (uuid.UUID, error)
	PayoutRun(ctx context.Context, runID uuid.UUID) (PayoutRun, error)
	ApprovePayoutRun(ctx context.Context, run PayoutRun, approverID uuid.UUID, bankDocRef string) (bool, error)
	VoucherForRefund(ctx context.Context, voucherID uuid.UUID) (VoucherRef, error)
	RequestRefund(ctx context.Context, voucher VoucherRef, requesterID uuid.UUID, reason string, auto bool) (uuid.UUID, error)
	RefundPaid(ctx context.Context, refundID uuid.UUID, approverID uuid.UUID) (bool, error)
	OpenDispute(ctx context.Context, voucherID, openerID uuid.UUID, reason string) (bool, error)
	ExpiredForBreakage(ctx context.Context, cutoff time.Time, batch int) ([]BreakageItem, error)
	ApplyBreakage(ctx context.Context, item BreakageItem) (bool, error)
	ActiveVouchers(ctx context.Context) ([]VoucherValue, error)
	CustomerPrepaidBalance(ctx context.Context) (int64, error)
	RecordFinding(ctx context.Context, check string, expected, actual int64) error
}

// Service applies the finance rules.
type Service struct {
	repo Repository
	now  func() time.Time
}

// NewService binds the finance rules to a repository.
func NewService(repo Repository, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{repo: repo, now: now}
}

// PreparePayout builds a payout run from the redemptions past the dispute window. Finance operators
// and superadmins prepare; a separate approver pays (four-eyes).
func (s *Service) PreparePayout(ctx context.Context, actor Actor, merchantID uuid.UUID) (uuid.UUID, error) {
	if !financeOperator(actor.Role) {
		return uuid.Nil, ErrForbidden
	}
	onHold, err := s.repo.MerchantOnHold(ctx, merchantID)
	if err != nil {
		return uuid.Nil, err
	}
	if onHold {
		return uuid.Nil, ErrMerchantOnHold
	}
	payable, err := s.repo.PayableRedemptions(ctx, merchantID, s.now().Add(-PayoutDisputeWindow))
	if err != nil {
		return uuid.Nil, err
	}
	var total int64
	ids := make([]uuid.UUID, 0, len(payable))
	for _, item := range payable {
		total += item.Net
		ids = append(ids, item.ID)
	}
	if total <= 0 {
		return uuid.Nil, ErrNothingToPay
	}
	return s.repo.CreatePayoutRun(ctx, merchantID, total, actor.UserID, ids)
}

// ApprovePayout sends the money. The approver must differ from the preparer (AT-18).
func (s *Service) ApprovePayout(ctx context.Context, actor Actor, runID uuid.UUID, bankDocRef string) error {
	if !financeApprover(actor.Role) {
		return ErrForbidden
	}
	run, err := s.repo.PayoutRun(ctx, runID)
	if err != nil {
		return err
	}
	if run.Status != "pending_approval" {
		return ErrNotPending
	}
	if run.PreparedBy == actor.UserID {
		return ErrSelfApproval
	}
	ok, err := s.repo.ApprovePayoutRun(ctx, run, actor.UserID, bankDocRef)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotPending
	}
	return nil
}

// RequestRefund refunds an unused voucher. Within 14 days of payment it is automatic; later requests
// wait for support and finance (FR-06).
func (s *Service) RequestRefund(ctx context.Context, buyerID, voucherID uuid.UUID, reason string) (uuid.UUID, error) {
	voucher, err := s.repo.VoucherForRefund(ctx, voucherID)
	if err != nil {
		return uuid.Nil, err
	}
	if voucher.BuyerID != buyerID {
		return uuid.Nil, ErrNotOwner
	}
	if voucher.Status != "active" || voucher.RedemptionsUsed > 0 {
		return uuid.Nil, ErrNotRefundable
	}
	auto := s.now().Sub(voucher.PaidAt) <= RefundWindow
	return s.repo.RequestRefund(ctx, voucher, buyerID, reason, auto)
}

// MarkRefundPaid records a refund that left the bank. The approver must differ from the requester.
func (s *Service) MarkRefundPaid(ctx context.Context, actor Actor, refundID uuid.UUID) error {
	if !financeApprover(actor.Role) {
		return ErrForbidden
	}
	ok, err := s.repo.RefundPaid(ctx, refundID, actor.UserID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return nil
}

// OpenDispute holds the payout of the buyer's latest redemption (BR-15).
func (s *Service) OpenDispute(ctx context.Context, buyerID, voucherID uuid.UUID, reason string) error {
	voucher, err := s.repo.VoucherForRefund(ctx, voucherID)
	if err != nil {
		return err
	}
	if voucher.BuyerID != buyerID {
		return ErrNotOwner
	}
	ok, err := s.repo.OpenDispute(ctx, voucherID, buyerID, reason)
	if err != nil {
		return err
	}
	if !ok {
		return ErrDisputeExists
	}
	return nil
}

// RunBreakage recognises value of vouchers that stayed unused 90 days after their window closed.
// The value moves to the expired account; it is not income. It returns how many vouchers moved.
func (s *Service) RunBreakage(ctx context.Context, batch int) (int, error) {
	cutoff := s.now().Add(-BreakageWindow)
	items, err := s.repo.ExpiredForBreakage(ctx, cutoff, batch)
	if err != nil {
		return 0, err
	}
	moved := 0
	for _, item := range items {
		ok, err := s.repo.ApplyBreakage(ctx, item)
		if err != nil {
			return moved, err
		}
		if ok {
			moved++
		}
	}
	return moved, nil
}

// Reconcile checks that the buyers' prepaid balance equals the unused value of active vouchers, and
// that the ledger balances overall. Mismatches are recorded as findings for finance to look at.
func (s *Service) Reconcile(ctx context.Context) (bool, error) {
	values, err := s.repo.ActiveVouchers(ctx)
	if err != nil {
		return false, err
	}
	var expected int64
	for _, value := range values {
		expected += value.Remaining
	}
	actual, err := s.repo.CustomerPrepaidBalance(ctx)
	if err != nil {
		return false, err
	}
	if expected == actual {
		return true, nil
	}
	return false, s.repo.RecordFinding(ctx, "customer_prepaid_equals_active_vouchers", expected, actual)
}

// Remaining returns the unused value of a voucher given how many shares were redeemed.
func Remaining(unitPrice int64, allowed, used int) int64 {
	shares, err := ledger.Split(unitPrice, allowed)
	if err != nil {
		return 0
	}
	remaining := unitPrice
	for i := 0; i < used && i < len(shares); i++ {
		remaining -= shares[i]
	}
	return remaining
}

func financeOperator(role string) bool {
	return role == "finance" || role == "superadmin"
}

func financeApprover(role string) bool {
	return role == "finance" || role == "superadmin"
}
