package moderation

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/grupon/grupon/services/api/internal/domain/catalog"
)

type fakeRepo struct {
	record   Record
	to       string
	reason   string
	audited  bool
	statusOK bool
}

func (f *fakeRepo) ListPendingIDs(context.Context, catalog.Page) ([]uuid.UUID, int64, error) {
	return nil, 0, nil
}

func (f *fakeRepo) Summaries(context.Context, []uuid.UUID) ([]catalog.DealSummary, error) {
	return nil, nil
}

func (f *fakeRepo) Record(context.Context, uuid.UUID) (Record, error) { return f.record, nil }

func (f *fakeRepo) Transition(_ context.Context, _ uuid.UUID, to, reason string, audit AuditEntry) error {
	f.to, f.reason, f.audited = to, reason, audit.Action != ""
	return nil
}

func (f *fakeRepo) Detail(context.Context, uuid.UUID) (catalog.Deal, error) {
	return catalog.Deal{}, nil
}

func pendingDeal(startsAt time.Time) *fakeRepo {
	return &fakeRepo{record: Record{ID: uuid.New(), MerchantID: uuid.New(), Status: statusInReview, StartsAt: startsAt}}
}

func TestFinanceCannotModerate(t *testing.T) {
	repo := pendingDeal(time.Now())
	service := NewService(repo, time.Now)
	_, err := service.Approve(context.Background(), Actor{UserID: uuid.New(), Role: "finance"}, repo.record.ID)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("finance approve: got %v", err)
	}
}

func TestApproveSchedulesFutureDealsAndActivatesCurrentOnes(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	actor := Actor{UserID: uuid.New(), Role: "moderator"}

	future := pendingDeal(now.Add(48 * time.Hour))
	if _, err := NewService(future, func() time.Time { return now }).Approve(context.Background(), actor, future.record.ID); err != nil {
		t.Fatal(err)
	}
	if future.to != statusScheduled || !future.audited {
		t.Fatalf("future deal: to=%s audited=%v", future.to, future.audited)
	}

	current := pendingDeal(now.Add(-time.Hour))
	if _, err := NewService(current, func() time.Time { return now }).Approve(context.Background(), actor, current.record.ID); err != nil {
		t.Fatal(err)
	}
	if current.to != statusActive {
		t.Fatalf("current deal: to=%s", current.to)
	}
}

func TestRejectNeedsReasonAndOnlyPendingDeals(t *testing.T) {
	actor := Actor{UserID: uuid.New(), Role: "admin"}
	repo := pendingDeal(time.Now())
	service := NewService(repo, time.Now)

	if _, err := service.Reject(context.Background(), actor, repo.record.ID, " "); err == nil {
		t.Fatal("empty reason accepted")
	}
	if _, err := service.Reject(context.Background(), actor, repo.record.ID, "Photos are blurry"); err != nil {
		t.Fatal(err)
	}
	if repo.to != statusChangesRequested || repo.reason != "Photos are blurry" {
		t.Fatalf("reject: to=%s reason=%q", repo.to, repo.reason)
	}

	repo.record.Status = statusActive
	if _, err := service.Reject(context.Background(), actor, repo.record.ID, "Too late"); !errors.Is(err, ErrNotPending) {
		t.Fatalf("reject active deal: got %v", err)
	}
}
