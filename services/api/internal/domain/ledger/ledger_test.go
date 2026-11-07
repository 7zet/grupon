package ledger

import (
	"errors"
	"testing"
)

const merchant = "m1"

func TestEveryPlanBalances(t *testing.T) {
	plans := map[string]Entry{
		"payment":     PaymentReceived("i1", merchant, 10000000),
		"redemption":  Redeemed("r1", merchant, 10000000, Commission(10000000, 1500)),
		"reversal":    Reversed("r1", merchant, 10000000, Commission(10000000, 1500)),
		"payout":      PayoutSent("p1", merchant, 8500000, 0),
		"psp":         PSPSettled("b1", 10000000, 200000),
		"refund":      RefundApproved("f1", merchant, 10000000),
		"refund-paid": RefundPaid("f1", 10000000),
		"breakage":    Expired("v1", merchant, 10000000),
	}
	for name, entry := range plans {
		if err := entry.Validate(); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestRedemptionSplitsCommission(t *testing.T) {
	entry := Redeemed("r1", merchant, 15000000, Commission(15000000, 1500))
	want := map[string]int64{
		CustomerPrepaid + merchant: 15000000,
		MerchantPayable + merchant: 12750000,
		CommissionRevenue:          2250000,
	}
	for _, line := range entry.Lines {
		if line.Direction == Debit && want[line.Account] != 0 && line.Amount != want[line.Account] {
			t.Errorf("%s debit = %d", line.Account, line.Amount)
		}
	}
	if len(entry.Lines) != 3 {
		t.Fatalf("lines = %d", len(entry.Lines))
	}
}

func TestSplitLosesNoTiyin(t *testing.T) {
	shares, err := Split(100000, 3)
	if err != nil {
		t.Fatal(err)
	}
	var sum int64
	for _, s := range shares {
		sum += s
	}
	if sum != 100000 || shares[2] != 33334 {
		t.Fatalf("shares = %v, sum = %d", shares, sum)
	}
}

func TestUnbalancedEntryIsRejected(t *testing.T) {
	bad := Entry{Kind: "x", Reference: "1", IdempotencyKey: "k", Lines: []Line{
		{Account: BankOperating, Direction: Debit, Amount: 100},
		{Account: PSPReceivable, Direction: Credit, Amount: 90},
	}}
	if err := bad.Validate(); !errors.Is(err, ErrUnbalanced) {
		t.Fatalf("got %v", err)
	}
	single := Entry{Kind: "x", Reference: "1", IdempotencyKey: "k", Lines: []Line{{Account: BankOperating, Direction: Debit, Amount: 100}}}
	if err := single.Validate(); !errors.Is(err, ErrUnbalanced) {
		t.Fatalf("single line: got %v", err)
	}
}
