package redemption

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/grupon/grupon/services/api/internal/domain/catalog"
	"github.com/grupon/grupon/services/api/internal/domain/commerce"
	"github.com/grupon/grupon/services/api/internal/domain/merchant"
)

type fakeRepo struct {
	redemption Redemption
	reversed   bool
	voucher    commerce.Voucher
	locations  map[uuid.UUID]bool
	attached   map[uuid.UUID]bool
	redeemed   bool
	audits     int
	denied     []string
}

// RecordDenied keeps the action of each refused scan so tests can check the audit trail.
func (f *fakeRepo) RecordDenied(_ context.Context, audit AuditEntry) error {
	f.denied = append(f.denied, audit.Action)
	return nil
}

func (f *fakeRepo) Find(context.Context, uuid.UUID) (commerce.Voucher, error) { return f.voucher, nil }

func (f *fakeRepo) FindByBackupCode(_ context.Context, _ string) (commerce.Voucher, error) {
	return commerce.Voucher{}, ErrNotFound
}

func (f *fakeRepo) FindByCode(_ context.Context, code string) (commerce.Voucher, error) {
	if code != f.voucher.Code {
		return commerce.Voucher{}, ErrNotFound
	}
	return f.voucher, nil
}

func (f *fakeRepo) LocationOwned(_ context.Context, _, locationID uuid.UUID) (bool, error) {
	return f.locations[locationID], nil
}

func (f *fakeRepo) LocationAttached(_ context.Context, _, locationID uuid.UUID) (bool, error) {
	return f.attached[locationID], nil
}

func (f *fakeRepo) Redeem(_ context.Context, _, _, _ uuid.UUID, at time.Time, audit AuditEntry, _ *Geo) (bool, error) {
	if f.voucher.Status != StatusActive || !at.Before(f.voucher.ValidUntil) {
		return false, nil
	}
	f.voucher.Status = StatusRedeemed
	f.redeemed = true
	f.audits++
	return true, nil
}

func (f *fakeRepo) View(_ context.Context, voucher commerce.Voucher) (commerce.VoucherView, error) {
	return commerce.VoucherView{Voucher: voucher}, nil
}

var now = time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)

func newFixture(status string, validUntil time.Time) (*fakeRepo, *Service, uuid.UUID, uuid.UUID, merchant.Actor) {
	merchantID := uuid.New()
	locationID := uuid.New()
	voucherID := uuid.New()
	repo := &fakeRepo{
		voucher:   commerce.Voucher{ID: voucherID, DealID: uuid.New(), MerchantID: merchantID, Code: "7KQ4-M9TX", Status: status, ValidFrom: now.AddDate(0, -1, 0), ValidUntil: validUntil},
		locations: map[uuid.UUID]bool{locationID: true},
		attached:  map[uuid.UUID]bool{locationID: true},
	}
	verify := func(token string) (uuid.UUID, error) {
		if token != "good" {
			return uuid.Nil, errors.New("bad token")
		}
		return voucherID, nil
	}
	return repo, NewService(repo, verify, func() time.Time { return now }), merchantID, locationID, merchant.Actor{UserID: uuid.New(), Role: merchant.RoleCashier}
}

func TestRedeemRequiresABranchAttachedToTheDeal(t *testing.T) {
	repo, service, merchantID, locationID, actor := newFixture(StatusActive, now.AddDate(0, 1, 0))
	repo.attached[locationID] = false
	_, err := service.Redeem(context.Background(), merchantID, actor, repo.voucher.ID, locationID, nil)
	var input *merchant.InvalidInputError
	if !errors.As(err, &input) || input.Field != "location_id" || input.Code != "location_not_attached_to_deal" {
		t.Fatalf("redeem at unattached branch: %v", err)
	}
	if repo.redeemed {
		t.Fatal("voucher was redeemed at a branch not attached to its deal")
	}
}

func TestLookupByCodeAndToken(t *testing.T) {
	repo, service, merchantID, _, _ := newFixture(StatusActive, now.AddDate(0, 1, 0))
	if _, err := service.Lookup(context.Background(), merchantID, uuid.New(), "", " 7kq4-m9tx "); err != nil {
		t.Fatalf("lookup by code: %v", err)
	}
	if _, err := service.Lookup(context.Background(), merchantID, uuid.New(), "good", ""); err != nil {
		t.Fatalf("lookup by token: %v", err)
	}
	if _, err := service.Lookup(context.Background(), merchantID, uuid.New(), "forged", ""); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("forged token: got %v", err)
	}
	if _, err := service.Lookup(context.Background(), uuid.New(), uuid.New(), "", repo.voucher.Code); !errors.Is(err, ErrWrongMerchant) {
		t.Fatalf("other merchant: got %v", err)
	}
}

func TestRedeemOnceAndReportStates(t *testing.T) {
	repo, service, merchantID, locationID, actor := newFixture(StatusActive, now.AddDate(0, 1, 0))
	view, err := service.Redeem(context.Background(), merchantID, actor, repo.voucher.ID, locationID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if view.Voucher.Status != StatusRedeemed || repo.audits != 1 {
		t.Fatalf("redeem: status=%s audits=%d", view.Voucher.Status, repo.audits)
	}
	if _, err := service.Redeem(context.Background(), merchantID, actor, repo.voucher.ID, locationID, nil); !errors.Is(err, ErrAlreadyRedeemed) {
		t.Fatalf("second redeem: got %v", err)
	}
}

// AT-04: a cashier of another business scans the voucher. The scan is refused, the voucher does not change, and an audit row is written.
func TestForeignMerchantScanIsRefusedAndAudited(t *testing.T) {
	repo, service, _, locationID, actor := newFixture(StatusActive, now.AddDate(0, 1, 0))
	otherMerchant := uuid.New()
	if _, err := service.Redeem(context.Background(), otherMerchant, actor, repo.voucher.ID, locationID, nil); !errors.Is(err, ErrWrongMerchant) {
		t.Fatalf("foreign scan: got %v, want ErrWrongMerchant", err)
	}
	if repo.redeemed || repo.audits != 0 {
		t.Fatalf("voucher changed by a refused scan: redeemed=%v audits=%d", repo.redeemed, repo.audits)
	}
	if len(repo.denied) != 1 || repo.denied[0] != "voucher_scan_denied" {
		t.Fatalf("denied audit rows = %v, want one voucher_scan_denied", repo.denied)
	}
}

func TestRedeemRejectsExpiredAndForeignLocation(t *testing.T) {
	repo, service, merchantID, locationID, actor := newFixture(StatusActive, now.Add(-time.Hour))
	if _, err := service.Redeem(context.Background(), merchantID, actor, repo.voucher.ID, locationID, nil); !errors.Is(err, ErrExpired) {
		t.Fatalf("expired voucher: got %v", err)
	}

	repo, service, merchantID, _, actor = newFixture(StatusActive, now.AddDate(0, 1, 0))
	_, err := service.Redeem(context.Background(), merchantID, actor, repo.voucher.ID, uuid.New(), nil)
	var invalid *merchant.InvalidInputError
	if !errors.As(err, &invalid) || invalid.Code != "location_not_found" {
		t.Fatalf("foreign location: got %v", err)
	}
}

func (f *fakeRepo) AttemptStats(context.Context, uuid.UUID, time.Time, time.Time) (AttemptCounts, error) {
	return AttemptCounts{}, nil
}

func (f *fakeRepo) RecordAttempt(context.Context, uuid.UUID, bool) error { return nil }

func (f *fakeRepo) LatestOpenRedemptionForVoucher(context.Context, uuid.UUID) (Redemption, error) {
	if f.redemption.ID == uuid.Nil {
		return Redemption{}, ErrNotFound
	}
	return f.redemption, nil
}

func (f *fakeRepo) ReversalCandidate(context.Context, uuid.UUID, uuid.UUID) (Redemption, error) {
	if f.redemption.ID == uuid.Nil {
		return Redemption{}, ErrNotFound
	}
	return f.redemption, nil
}

func (f *fakeRepo) ReverseUse(context.Context, uuid.UUID, Redemption, string, AuditEntry) error {
	f.reversed = true
	return nil
}
func (f *fakeRepo) History(context.Context, uuid.UUID, *uuid.UUID, catalog.Page) ([]HistoryItem, int64, error) {
	return nil, 0, nil
}

func TestCashierCanReverseOnlyWithinTheWindow(t *testing.T) {
	repo, service, merchantID, _, _ := newFixture(StatusRedeemed, now.AddDate(0, 1, 0))
	cashier := merchant.Actor{UserID: uuid.New(), Role: merchant.RoleCashier}
	repo.redemption = Redemption{ID: uuid.New(), VoucherID: repo.voucher.ID, StaffID: cashier.UserID, CreatedAt: now.Add(-5 * time.Minute)}
	if err := service.Reverse(context.Background(), merchantID, cashier, repo.voucher.ID, "wrong scan"); err != nil || !repo.reversed {
		t.Fatalf("inside window: err=%v reversed=%v", err, repo.reversed)
	}

	repo.reversed = false
	repo.redemption.CreatedAt = now.Add(-11 * time.Minute)
	if err := service.Reverse(context.Background(), merchantID, cashier, repo.voucher.ID, "late"); !errors.Is(err, ErrReversalWindowClosed) {
		t.Fatalf("after window: got %v", err)
	}

	// The window binds the owner too: after 10 minutes only support can reverse.
	owner := merchant.Actor{UserID: uuid.New(), Role: merchant.RoleOwner}
	repo.reversed = false
	if err := service.Reverse(context.Background(), merchantID, owner, repo.voucher.ID, "owner fix"); !errors.Is(err, ErrReversalWindowClosed) || repo.reversed {
		t.Fatalf("owner after window: err=%v reversed=%v", err, repo.reversed)
	}
	if err := service.SupportReverse(context.Background(), uuid.New(), "support", repo.voucher.ID, "wrong customer"); err != nil || !repo.reversed {
		t.Fatalf("support after window: err=%v reversed=%v", err, repo.reversed)
	}
	if err := service.SupportReverse(context.Background(), uuid.New(), "admin", repo.voucher.ID, "not allowed"); !errors.Is(err, merchant.ErrForbidden) {
		t.Fatalf("non-support reversal: got %v", err)
	}
}

// limitedRepo reports attempt counts so the scanner limits can be tested without a database.
type limitedRepo struct {
	*fakeRepo
	counts AttemptCounts
}

func (l limitedRepo) AttemptStats(context.Context, uuid.UUID, time.Time, time.Time) (AttemptCounts, error) {
	return l.counts, nil
}

func TestScannerLimitsSlowBurstsAndBlocksFailedCodes(t *testing.T) {
	repo, service, merchantID, _, _ := newFixture(StatusActive, now.AddDate(0, 1, 0))
	staff := uuid.New()

	burst := NewService(limitedRepo{fakeRepo: repo, counts: AttemptCounts{LastMinute: LookupsPerMinute}}, nil, func() time.Time { return now })
	if _, err := burst.Lookup(context.Background(), merchantID, staff, "", repo.voucher.Code); !errors.Is(err, ErrTooManyAttempts) {
		t.Fatalf("31st lookup in a minute: got %v, want ErrTooManyAttempts", err)
	}

	blocked := NewService(limitedRepo{fakeRepo: repo, counts: AttemptCounts{FailedRecent: FailedLookupLimit}}, nil, func() time.Time { return now })
	if _, err := blocked.Lookup(context.Background(), merchantID, staff, "", repo.voucher.Code); !errors.Is(err, ErrScannerBlocked) {
		t.Fatalf("after ten failed codes: got %v, want ErrScannerBlocked", err)
	}

	if _, err := service.Lookup(context.Background(), merchantID, staff, "", repo.voucher.Code); err != nil {
		t.Fatalf("normal lookup: %v", err)
	}
}
