package redemption

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/grupon/grupon/services/api/internal/domain/commerce"
	"github.com/grupon/grupon/services/api/internal/domain/merchant"
)

type fakeRepo struct {
	redemption Redemption
	reversed   bool
	voucher    commerce.Voucher
	locations  map[uuid.UUID]bool
	redeemed   bool
	audits     int
}

func (f *fakeRepo) Find(context.Context, uuid.UUID) (commerce.Voucher, error) { return f.voucher, nil }

func (f *fakeRepo) FindByCode(_ context.Context, code string) (commerce.Voucher, error) {
	if code != f.voucher.Code {
		return commerce.Voucher{}, ErrNotFound
	}
	return f.voucher, nil
}

func (f *fakeRepo) LocationOwned(_ context.Context, _, locationID uuid.UUID) (bool, error) {
	return f.locations[locationID], nil
}

func (f *fakeRepo) Redeem(_ context.Context, _, _, _ uuid.UUID, at time.Time, audit AuditEntry) (bool, error) {
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
		voucher:   commerce.Voucher{ID: voucherID, MerchantID: merchantID, Code: "7KQ4-M9TX", Status: status, ValidFrom: now.AddDate(0, -1, 0), ValidUntil: validUntil},
		locations: map[uuid.UUID]bool{locationID: true},
	}
	verify := func(token string) (uuid.UUID, error) {
		if token != "good" {
			return uuid.Nil, errors.New("bad token")
		}
		return voucherID, nil
	}
	return repo, NewService(repo, verify, func() time.Time { return now }), merchantID, locationID, merchant.Actor{UserID: uuid.New(), Role: merchant.RoleCashier}
}

func TestLookupByCodeAndToken(t *testing.T) {
	repo, service, merchantID, _, _ := newFixture(StatusActive, now.AddDate(0, 1, 0))
	if _, err := service.Lookup(context.Background(), merchantID, "", " 7kq4-m9tx "); err != nil {
		t.Fatalf("lookup by code: %v", err)
	}
	if _, err := service.Lookup(context.Background(), merchantID, "good", ""); err != nil {
		t.Fatalf("lookup by token: %v", err)
	}
	if _, err := service.Lookup(context.Background(), merchantID, "forged", ""); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("forged token: got %v", err)
	}
	if _, err := service.Lookup(context.Background(), uuid.New(), "", repo.voucher.Code); !errors.Is(err, ErrWrongMerchant) {
		t.Fatalf("other merchant: got %v", err)
	}
}

func TestRedeemOnceAndReportStates(t *testing.T) {
	repo, service, merchantID, locationID, actor := newFixture(StatusActive, now.AddDate(0, 1, 0))
	view, err := service.Redeem(context.Background(), merchantID, actor, repo.voucher.ID, locationID)
	if err != nil {
		t.Fatal(err)
	}
	if view.Voucher.Status != StatusRedeemed || repo.audits != 1 {
		t.Fatalf("redeem: status=%s audits=%d", view.Voucher.Status, repo.audits)
	}
	if _, err := service.Redeem(context.Background(), merchantID, actor, repo.voucher.ID, locationID); !errors.Is(err, ErrAlreadyRedeemed) {
		t.Fatalf("second redeem: got %v", err)
	}
}

func TestRedeemRejectsExpiredAndForeignLocation(t *testing.T) {
	repo, service, merchantID, locationID, actor := newFixture(StatusActive, now.Add(-time.Hour))
	if _, err := service.Redeem(context.Background(), merchantID, actor, repo.voucher.ID, locationID); !errors.Is(err, ErrExpired) {
		t.Fatalf("expired voucher: got %v", err)
	}

	repo, service, merchantID, _, actor = newFixture(StatusActive, now.AddDate(0, 1, 0))
	_, err := service.Redeem(context.Background(), merchantID, actor, repo.voucher.ID, uuid.New())
	var invalid *merchant.InvalidInputError
	if !errors.As(err, &invalid) || invalid.Code != "location_not_found" {
		t.Fatalf("foreign location: got %v", err)
	}
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

	owner := merchant.Actor{UserID: uuid.New(), Role: merchant.RoleOwner}
	if err := service.Reverse(context.Background(), merchantID, owner, repo.voucher.ID, "owner fix"); err != nil || !repo.reversed {
		t.Fatalf("owner any time: err=%v reversed=%v", err, repo.reversed)
	}
}
