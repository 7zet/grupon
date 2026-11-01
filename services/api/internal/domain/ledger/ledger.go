// Package ledger turns money events into balanced journal entries (Dealo PRD §8, ADR-0005).
// Entries are append-only; a mistake is corrected by a reversing entry, never by editing.
// Amounts are int64 tiyin. Division rounds down, and the remainder goes to the last share.
package ledger

import (
	"errors"
	"fmt"
)

// Account codes. Per-merchant accounts carry the merchant id after the colon.
const (
	PSPReceivable      = "psp_receivable"
	BankOperating      = "bank:operating"
	PSPFees            = "psp_fees"
	CommissionRevenue  = "commission_revenue"
	RefundsPayable     = "refunds_payable"
	BreakageRevenue    = "breakage_revenue"
	CustomerPrepaid    = "customer_prepaid:"
	CustomerPrepaidExp = "customer_prepaid_expired:"
	MerchantPayable    = "merchant_payable:"
)

// Direction is the side of a line.
type Direction string

const (
	Debit  Direction = "debit"
	Credit Direction = "credit"
)

// Line is one side of a journal entry.
type Line struct {
	Account   string
	Direction Direction
	Amount    int64
}

// Entry is a balanced set of lines written under one idempotency key.
type Entry struct {
	Kind           string
	Reference      string
	IdempotencyKey string
	Lines          []Line
}

// ErrUnbalanced reports an entry whose debits and credits differ, or that has fewer than two lines.
var ErrUnbalanced = errors.New("ledger: entry is not balanced")

// Validate checks the rules the database enforces too, so a bad plan fails before any write.
func (e Entry) Validate() error {
	if e.IdempotencyKey == "" || e.Kind == "" || e.Reference == "" {
		return errors.New("ledger: kind, reference and idempotency key are required")
	}
	if len(e.Lines) < 2 {
		return ErrUnbalanced
	}
	var debits, credits int64
	for _, line := range e.Lines {
		if line.Amount <= 0 {
			return fmt.Errorf("ledger: line amount must be positive, got %d", line.Amount)
		}
		switch line.Direction {
		case Debit:
			debits += line.Amount
		case Credit:
			credits += line.Amount
		default:
			return fmt.Errorf("ledger: unknown direction %q", line.Direction)
		}
	}
	if debits != credits {
		return ErrUnbalanced
	}
	return nil
}

// Split divides total into n shares that add up exactly. Every share is total/n rounded down; the
// last share takes the remainder, so no tiyin is lost (GAP-07). The shares are returned in order.
func Split(total int64, n int) ([]int64, error) {
	if n < 1 || total < 0 {
		return nil, errors.New("ledger: split needs a positive count and a non-negative total")
	}
	shares := make([]int64, n)
	base := total / int64(n)
	for i := range shares {
		shares[i] = base
	}
	shares[n-1] += total - base*int64(n)
	return shares, nil
}

// Commission returns the platform's cut of an amount, rounded down to whole tiyin.
func Commission(amount int64, basisPoints int) int64 {
	return amount * int64(basisPoints) / 10000
}

// PaymentReceived records money the customer paid for one order item through the PSP. The platform
// holds it as a receivable from the PSP and owes it back to the customer as prepaid value.
func PaymentReceived(itemID string, merchantID string, amount int64) Entry {
	return Entry{
		Kind:           "payment_received",
		Reference:      itemID,
		IdempotencyKey: "payment_received:" + itemID,
		Lines: []Line{
			{Account: PSPReceivable, Direction: Debit, Amount: amount},
			{Account: CustomerPrepaid + merchantID, Direction: Credit, Amount: amount},
		},
	}
}

// Redeemed moves the redeemed share from prepaid value to what the merchant is owed, minus the
// platform commission.
func Redeemed(redemptionID, merchantID string, amount int64, commission int64) Entry {
	return Entry{
		Kind:           "redemption",
		Reference:      redemptionID,
		IdempotencyKey: "redemption:" + redemptionID,
		Lines: withoutZero(
			Line{Account: CustomerPrepaid + merchantID, Direction: Debit, Amount: amount},
			Line{Account: MerchantPayable + merchantID, Direction: Credit, Amount: amount - commission},
			Line{Account: CommissionRevenue, Direction: Credit, Amount: commission},
		),
	}
}

// Reversed undoes a redemption exactly (a reversing entry, never an edit).
func Reversed(redemptionID, merchantID string, amount int64, commission int64) Entry {
	return Entry{
		Kind:           "redemption_reversal",
		Reference:      redemptionID,
		IdempotencyKey: "redemption_reversal:" + redemptionID,
		Lines: withoutZero(
			Line{Account: MerchantPayable + merchantID, Direction: Debit, Amount: amount - commission},
			Line{Account: CommissionRevenue, Direction: Debit, Amount: commission},
			Line{Account: CustomerPrepaid + merchantID, Direction: Credit, Amount: amount},
		),
	}
}

// PayoutSent records a payout that left the bank.
func PayoutSent(runID, merchantID string, amount int64) Entry {
	return Entry{
		Kind:           "payout_sent",
		Reference:      runID,
		IdempotencyKey: "payout_sent:" + runID,
		Lines: []Line{
			{Account: MerchantPayable + merchantID, Direction: Debit, Amount: amount},
			{Account: BankOperating, Direction: Credit, Amount: amount},
		},
	}
}

// PSPSettled records the PSP paying the platform for a batch, net of its fee.
func PSPSettled(batchID string, gross, fee int64) Entry {
	return Entry{
		Kind:           "psp_settlement",
		Reference:      batchID,
		IdempotencyKey: "psp_settlement:" + batchID,
		Lines: withoutZero(
			Line{Account: BankOperating, Direction: Debit, Amount: gross - fee},
			Line{Account: PSPFees, Direction: Debit, Amount: fee},
			Line{Account: PSPReceivable, Direction: Credit, Amount: gross},
		),
	}
}

// RefundApproved moves unused prepaid value into the refund obligation.
func RefundApproved(refundID, merchantID string, amount int64) Entry {
	return Entry{
		Kind:           "refund_approved",
		Reference:      refundID,
		IdempotencyKey: "refund_approved:" + refundID,
		Lines: []Line{
			{Account: CustomerPrepaid + merchantID, Direction: Debit, Amount: amount},
			{Account: RefundsPayable, Direction: Credit, Amount: amount},
		},
	}
}

// RefundPaid records the refund leaving the bank.
func RefundPaid(refundID string, amount int64) Entry {
	return Entry{
		Kind:           "refund_paid",
		Reference:      refundID,
		IdempotencyKey: "refund_paid:" + refundID,
		Lines: []Line{
			{Account: RefundsPayable, Direction: Debit, Amount: amount},
			{Account: BankOperating, Direction: Credit, Amount: amount},
		},
	}
}

// Expired moves value of a voucher that passed the 90-day refund window into the expired account
// (breakage, PRD §8.5). It is not recognised as revenue.
func Expired(voucherID, merchantID string, amount int64) Entry {
	return Entry{
		Kind:           "breakage",
		Reference:      voucherID,
		IdempotencyKey: "breakage:" + voucherID,
		Lines: []Line{
			{Account: CustomerPrepaid + merchantID, Direction: Debit, Amount: amount},
			{Account: CustomerPrepaidExp + merchantID, Direction: Credit, Amount: amount},
		},
	}
}

func withoutZero(lines ...Line) []Line {
	out := make([]Line, 0, len(lines))
	for _, line := range lines {
		if line.Amount != 0 {
			out = append(out, line)
		}
	}
	return out
}
