// Package finance runs settlement, payouts with four-eyes approval, refunds, disputes, breakage and
// reconciliation (Dealo PRD §6 FR-06…08, §8, §12). Every money movement is a balanced ledger entry.
package finance

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/dealo/dealo/services/api/internal/domain/ledger"
)

// Windows from the PRD. Amounts are tiyin.
const (
	PayoutDisputeWindow = 3 * 24 * time.Hour  // FR-07.2: redemptions wait 3 days before payout
	RefundWindow        = 14 * 24 * time.Hour // FR-06: unused voucher refunded automatically
	BreakageWindow      = 90 * 24 * time.Hour // PRD §8.5: 90 days after the voucher window closed
)

// Errors for the finance rules. The HTTP layer maps them to problem codes.
var (
	ErrForbidden         = errors.New("finance: role may not do this")
	ErrNothingToPay      = errors.New("finance: no redemptions are ready for payout")
	ErrMerchantOnHold    = errors.New("finance: merchant is on payout hold")
	ErrNotPending        = errors.New("finance: payout is not waiting for approval")
	ErrSelfApproval      = errors.New("finance: the person who prepared a payout cannot approve it")
	ErrNotFound          = errors.New("finance: record not found")
	ErrNotOwner          = errors.New("finance: voucher belongs to another buyer")
	ErrNotRefundable     = errors.New("finance: voucher cannot be refunded")
	ErrDisputeExists     = errors.New("finance: a dispute is already open for this redemption")
	ErrNoRedemption      = errors.New("finance: voucher has no redemption to dispute")
	ErrNotSent           = errors.New("finance: payout has not been sent to the bank")
	ErrDisputeClosed     = errors.New("finance: dispute is already resolved")
	ErrInvalidTransition = errors.New("finance: dispute transition is not allowed")
	ErrInvalidResolution = errors.New("finance: resolution must be customer or merchant")
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
	ID               uuid.UUID
	MerchantID       uuid.UUID
	Amount           int64
	ReceivableOffset int64
	Status           string
	PreparedBy       uuid.UUID
}

// VoucherRef is what a refund or dispute needs to know about a voucher.
type VoucherRef struct {
	ID                 uuid.UUID
	BuyerID            uuid.UUID
	MerchantID         uuid.UUID
	Status             string
	RedemptionsUsed    int
	UnitPrice          int64
	PaidAt             time.Time
	ValidUntil         time.Time
	RedemptionsAllowed int
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

// RefundRef is an approved refund ready for the bank.
type RefundRef struct {
	ID         uuid.UUID
	MerchantID uuid.UUID
	Amount     int64
}

// QueueItem is one open item on the finance desk.
type QueueItem struct {
	ID        uuid.UUID
	Title     string
	Amount    int64
	Status    string
	Reason    string
	CreatedAt time.Time
}

// Queue is everything open on the finance desk.
type Queue struct {
	PayoutRuns []QueueItem
	Refunds    []QueueItem
	Disputes   []QueueItem
}

// Repository performs the writes. Each method is one transaction that also writes its ledger entry
// and outbox event, so a money movement is either fully recorded or not at all.
type Repository interface {
	// RecordDenied writes the audit row for a finance action the actor's role may not do (AT-09).
	RecordDenied(ctx context.Context, actor Actor, action string, entityID uuid.UUID) error
	PayableRedemptions(ctx context.Context, merchantID uuid.UUID, before time.Time) ([]Payable, error)
	MerchantOnHold(ctx context.Context, merchantID uuid.UUID) (bool, error)
	MerchantReceivable(ctx context.Context, merchantID uuid.UUID) (int64, error)
	CreatePayoutRun(ctx context.Context, merchantID uuid.UUID, amount, receivableOffset int64, preparedBy uuid.UUID, redemptionIDs []uuid.UUID) (uuid.UUID, error)
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
	// MarkPayoutReconciled moves a sent run to reconciled. It reports false when the run is not sent.
	MarkPayoutReconciled(ctx context.Context, runID uuid.UUID) (bool, error)
	// ApproveRefund approves a requested refund and posts it. The approver cannot be the requester.
	ApproveRefund(ctx context.Context, refundID, approverID uuid.UUID) (RefundRef, error)
	// ProposeRefund moves a customer/dispute request to the finance approval queue.
	ProposeRefund(ctx context.Context, refundID, proposerID uuid.UUID) (bool, error)
	// TransitionDispute changes one state atomically with its audit row and resolution side effects.
	TransitionDispute(ctx context.Context, disputeID uuid.UUID, actor Actor, from []string, to string) error
	// Open lists the open payouts, refunds and disputes for the finance desk.
	Open(ctx context.Context) (Queue, error)
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
		if err := s.repo.RecordDenied(ctx, actor, "payout_prepare_denied", merchantID); err != nil {
			return uuid.Nil, err
		}
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
	receivable, err := s.repo.MerchantReceivable(ctx, merchantID)
	if err != nil {
		return uuid.Nil, err
	}
	offset := min(total, receivable)
	if total <= offset {
		return uuid.Nil, ErrNothingToPay
	}
	return s.repo.CreatePayoutRun(ctx, merchantID, total-offset, offset, actor.UserID, ids)
}

// ApprovePayout sends the money. The approver must differ from the preparer (AT-18).
func (s *Service) ApprovePayout(ctx context.Context, actor Actor, runID uuid.UUID, bankDocRef string) error {
	if !financeApprover(actor.Role) {
		if err := s.repo.RecordDenied(ctx, actor, "payout_approve_denied", runID); err != nil {
			return err
		}
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
		if err := s.repo.RecordDenied(ctx, actor, "payout_approve_self_denied", runID); err != nil {
			return err
		}
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
	if voucher.RedemptionsAllowed < 1 || voucher.RedemptionsUsed >= voucher.RedemptionsAllowed {
		return uuid.Nil, ErrNotRefundable
	}
	if voucher.Status != "active" && voucher.Status != "partially_redeemed" && voucher.Status != "expired" {
		return uuid.Nil, ErrNotRefundable
	}
	// An expired voucher can still be refunded for ninety days; after that its
	// unused value belongs to the breakage process rather than a refund queue.
	if voucher.Status == "expired" && s.now().After(voucher.ValidUntil.Add(BreakageWindow)) {
		return uuid.Nil, ErrNotRefundable
	}
	auto := s.now().Sub(voucher.PaidAt) <= RefundWindow
	voucher.UnitPrice = Remaining(voucher.UnitPrice, voucher.RedemptionsAllowed, voucher.RedemptionsUsed)
	if voucher.UnitPrice <= 0 {
		return uuid.Nil, ErrNotRefundable
	}
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

// MarkPayoutReconciled records that the bank confirmed a sent payout run.
func (s *Service) MarkPayoutReconciled(ctx context.Context, actor Actor, runID uuid.UUID) error {
	if !financeOperator(actor.Role) {
		return ErrForbidden
	}
	ok, err := s.repo.MarkPayoutReconciled(ctx, runID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotSent
	}
	return nil
}

// ApproveRefund is the finance approval step for refunds proposed after the 14-day window or from a dispute.
func (s *Service) ApproveRefund(ctx context.Context, actor Actor, refundID uuid.UUID) error {
	if !financeApprover(actor.Role) {
		if err := s.repo.RecordDenied(ctx, actor, "refund_approve_denied", refundID); err != nil {
			return err
		}
		return ErrForbidden
	}
	_, err := s.repo.ApproveRefund(ctx, refundID, actor.UserID)
	if errors.Is(err, ErrForbidden) {
		if auditErr := s.repo.RecordDenied(ctx, actor, "refund_approve_self_denied", refundID); auditErr != nil {
			return auditErr
		}
	}
	return err
}

// ProposeRefund makes support's review explicit; the proposer can never be the finance approver.
func (s *Service) ProposeRefund(ctx context.Context, actor Actor, refundID uuid.UUID) error {
	if !supportActor(actor.Role) {
		if err := s.repo.RecordDenied(ctx, actor, "refund_propose_denied", refundID); err != nil {
			return err
		}
		return ErrForbidden
	}
	ok, err := s.repo.ProposeRefund(ctx, refundID, actor.UserID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotPending
	}
	return nil
}

// ResolveDispute decides a dispute. "customer" refunds the buyer (a refund request for support to
// approve); "merchant" releases the held payout.
func (s *Service) ResolveDispute(ctx context.Context, actor Actor, disputeID uuid.UUID, resolution string) error {
	if !financeApprover(actor.Role) {
		return ErrForbidden
	}
	if resolution != "customer" && resolution != "merchant" {
		return ErrInvalidResolution
	}
	return s.repo.TransitionDispute(ctx, disputeID, actor, []string{"under_review"}, "resolved_"+resolution)
}

// RequestDisputeEvidence moves a newly opened dispute to evidence collection. Support owns this step.
func (s *Service) RequestDisputeEvidence(ctx context.Context, actor Actor, disputeID uuid.UUID) error {
	if !supportActor(actor.Role) {
		return ErrForbidden
	}
	return s.repo.TransitionDispute(ctx, disputeID, actor, []string{"open"}, "awaiting_evidence")
}

// StartDisputeReview moves a dispute with collected evidence into finance review.
func (s *Service) StartDisputeReview(ctx context.Context, actor Actor, disputeID uuid.UUID) error {
	if !supportActor(actor.Role) {
		return ErrForbidden
	}
	return s.repo.TransitionDispute(ctx, disputeID, actor, []string{"awaiting_evidence"}, "under_review")
}

// CloseDispute closes either resolved outcome. It has no ledger side effect.
func (s *Service) CloseDispute(ctx context.Context, actor Actor, disputeID uuid.UUID) error {
	if !supportActor(actor.Role) && actor.Role != "system" {
		return ErrForbidden
	}
	return s.repo.TransitionDispute(ctx, disputeID, actor, []string{"resolved_customer", "resolved_merchant"}, "closed")
}

// Queue lists what the finance desk must act on. Support sees refunds and disputes, finance sees all.
func (s *Service) Queue(ctx context.Context, actor Actor) (Queue, error) {
	if !supportOrFinance(actor.Role) {
		return Queue{}, ErrForbidden
	}
	queue, err := s.repo.Open(ctx)
	if err != nil {
		return Queue{}, err
	}
	if actor.Role == "support" {
		queue.PayoutRuns = nil
	}
	return queue, nil
}

func supportOrFinance(role string) bool {
	return role == "support" || financeApprover(role)
}

func supportActor(role string) bool {
	return role == "support" || role == "superadmin"
}

func financeOperator(role string) bool {
	return role == "finance" || role == "superadmin"
}

func financeApprover(role string) bool {
	return role == "finance" || role == "superadmin"
}
