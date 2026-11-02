package merchant

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Review errors. The HTTP layer maps them to problem codes.
var (
	ErrBusinessNotFound    = errors.New("merchant: business not found")
	ErrInvalidTransition   = errors.New("merchant: business is not in a state that allows this decision")
	ErrMerchantNotApproved = errors.New("merchant: business must be approved before it can publish deals")
)

// ReviewActor is a platform staff member deciding on a business.
type ReviewActor struct {
	UserID uuid.UUID
	Role   string
}

// reviewerRoles may review businesses, like deal moderators (FR-01.3).
var reviewerRoles = map[string]bool{"superadmin": true, "admin": true, "moderator": true}

// ReviewItem is one business in the review queue.
type ReviewItem struct {
	ID              uuid.UUID
	LegalName       string
	DisplayName     string
	STIR            string
	OrgForm         string
	TaxRegime       string
	BankMFO         string
	BankAccount     string
	ResponsibleName string
	ReviewStatus    string
	SubmittedAt     time.Time
	// Documents lists the kinds already uploaded: certificate and bank_letter.
	Documents []string
}

// ReviewAudit describes the decision for the audit trail.
type ReviewAudit struct {
	ActorID    uuid.UUID
	Role       string
	Action     string
	MerchantID uuid.UUID
	After      map[string]any
}

// ReviewRepository persists review decisions. Transition only moves a business whose current status is
// one of from; it returns ErrBusinessNotFound or ErrInvalidTransition otherwise.
type ReviewRepository interface {
	ListForReview(ctx context.Context, status string) ([]ReviewItem, error)
	Transition(ctx context.Context, merchantID uuid.UUID, from []string, to, reason string, audit ReviewAudit) error
}

// ReviewService applies the business review rules: submitted → in_review → approved, or rejected.
type ReviewService struct {
	repo ReviewRepository
}

// NewReviewService binds the review rules to a repository.
func NewReviewService(repo ReviewRepository) *ReviewService {
	return &ReviewService{repo: repo}
}

// List returns the queue. An empty status lists both submitted and in-review businesses.
func (s *ReviewService) List(ctx context.Context, actor ReviewActor, status string) ([]ReviewItem, error) {
	if !reviewerRoles[actor.Role] {
		return nil, ErrForbidden
	}
	return s.repo.ListForReview(ctx, status)
}

// Start moves a submitted business to in_review, so one reviewer owns the case.
func (s *ReviewService) Start(ctx context.Context, actor ReviewActor, merchantID uuid.UUID) error {
	return s.decide(ctx, actor, merchantID, []string{"submitted"}, "in_review", "", "merchant_review_started")
}

// Approve lets the business create deals. Only a business in review can be approved.
func (s *ReviewService) Approve(ctx context.Context, actor ReviewActor, merchantID uuid.UUID) error {
	return s.decide(ctx, actor, merchantID, []string{"in_review"}, "approved", "", "merchant_approved")
}

// Reject closes the application with a reason the merchant can read.
func (s *ReviewService) Reject(ctx context.Context, actor ReviewActor, merchantID uuid.UUID, reason string) error {
	reason = strings.TrimSpace(reason)
	if n := len([]rune(reason)); n < 3 || n > 500 {
		return &InvalidInputError{Field: "reason", Code: "length"}
	}
	return s.decide(ctx, actor, merchantID, []string{"submitted", "in_review"}, "rejected", reason, "merchant_rejected")
}

func (s *ReviewService) decide(ctx context.Context, actor ReviewActor, merchantID uuid.UUID, from []string, to, reason, action string) error {
	if !reviewerRoles[actor.Role] {
		return ErrForbidden
	}
	audit := ReviewAudit{
		ActorID:    actor.UserID,
		Role:       actor.Role,
		Action:     action,
		MerchantID: merchantID,
		After:      map[string]any{"status": to},
	}
	if reason != "" {
		audit.After["reason"] = reason
	}
	return s.repo.Transition(ctx, merchantID, from, to, reason, audit)
}
