package merchant

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

// memReviewRepo keeps one business and enforces the from-states like the SQL does.
type memReviewRepo struct {
	status  string
	audited []string
}

func (m *memReviewRepo) ListForReview(context.Context, string) ([]ReviewItem, error) {
	return []ReviewItem{{ReviewStatus: m.status}}, nil
}

func (m *memReviewRepo) Transition(_ context.Context, _ uuid.UUID, from []string, to, _ string, audit ReviewAudit) error {
	for _, f := range from {
		if m.status == f {
			m.status = to
			m.audited = append(m.audited, audit.Action)
			return nil
		}
	}
	return ErrInvalidTransition
}

var (
	reviewer = ReviewActor{UserID: uuid.New(), Role: "moderator"}
	business = uuid.New()
)

func TestBusinessMovesThroughReviewAndIsAudited(t *testing.T) {
	t.Parallel()
	repo := &memReviewRepo{status: "submitted"}
	service := NewReviewService(repo)
	ctx := context.Background()

	if err := service.Approve(ctx, reviewer, business); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("approve before review: got %v, want ErrInvalidTransition", err)
	}
	if err := service.Start(ctx, reviewer, business); err != nil {
		t.Fatalf("start review: %v", err)
	}
	if err := service.Approve(ctx, reviewer, business); err != nil {
		t.Fatalf("approve: %v", err)
	}
	if repo.status != "approved" {
		t.Fatalf("status = %s, want approved", repo.status)
	}
	if len(repo.audited) != 2 || repo.audited[1] != "merchant_approved" {
		t.Fatalf("audit trail = %v", repo.audited)
	}
}

func TestRejectNeedsAReasonAndAClosedBusinessCannotBeRejectedAgain(t *testing.T) {
	t.Parallel()
	repo := &memReviewRepo{status: "in_review"}
	service := NewReviewService(repo)
	ctx := context.Background()

	var invalid *InvalidInputError
	if err := service.Reject(ctx, reviewer, business, " "); !errors.As(err, &invalid) || invalid.Field != "reason" {
		t.Fatalf("empty reason: got %v", err)
	}
	if err := service.Reject(ctx, reviewer, business, "Bank rekvizitlari mos emas"); err != nil {
		t.Fatalf("reject: %v", err)
	}
	if err := service.Reject(ctx, reviewer, business, "Yana bir sabab"); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("second reject: got %v, want ErrInvalidTransition", err)
	}
}

func TestOnlyPlatformReviewersDecide(t *testing.T) {
	t.Parallel()
	repo := &memReviewRepo{status: "submitted"}
	service := NewReviewService(repo)
	cashier := ReviewActor{UserID: uuid.New(), Role: "finance"}
	if _, err := service.List(context.Background(), cashier, ""); !errors.Is(err, ErrForbidden) {
		t.Fatalf("list by finance: got %v, want ErrForbidden", err)
	}
	if err := service.Start(context.Background(), cashier, business); !errors.Is(err, ErrForbidden) {
		t.Fatalf("start by finance: got %v, want ErrForbidden", err)
	}
	if repo.status != "submitted" {
		t.Fatal("a forbidden decision changed the business")
	}
}

// notApprovedDealRepo reports a business that has not passed review.
type notApprovedDealRepo struct {
	fakeDealRepo
}

func (*notApprovedDealRepo) ReviewStatus(context.Context, uuid.UUID) (string, error) {
	return "submitted", nil
}

func TestDealCreationWaitsForApproval(t *testing.T) {
	t.Parallel()
	service := NewDealService(&notApprovedDealRepo{}, nil)
	_, err := service.Create(context.Background(), uuid.New(), Actor{UserID: uuid.New(), Role: "merchant_owner"}, validDeal())
	if !errors.Is(err, ErrMerchantNotApproved) {
		t.Fatalf("create before approval: got %v, want ErrMerchantNotApproved", err)
	}
}

func TestSuspendReinstateAndTerminate(t *testing.T) {
	t.Parallel()
	repo := &memReviewRepo{status: "approved"}
	service := NewReviewService(repo)
	ctx := context.Background()

	if err := service.Suspend(ctx, reviewer, business, "  "); err == nil {
		t.Fatal("suspend without reason succeeded")
	}
	if err := service.Suspend(ctx, reviewer, business, "Repeated customer complaints"); err != nil || repo.status != "suspended" {
		t.Fatalf("suspend: err=%v status=%s", err, repo.status)
	}
	if err := service.Reinstate(ctx, reviewer, business); err != nil || repo.status != "approved" {
		t.Fatalf("reinstate: err=%v status=%s", err, repo.status)
	}
	if err := service.Reinstate(ctx, reviewer, business); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("reinstate approved business: got %v", err)
	}
	if err := service.Terminate(ctx, reviewer, business, "Fraud confirmed"); err != nil || repo.status != "terminated" {
		t.Fatalf("terminate: err=%v status=%s", err, repo.status)
	}
	if err := service.Suspend(ctx, reviewer, business, "Again after closing"); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("suspend terminated business: got %v", err)
	}
}
