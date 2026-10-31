// Package moderation holds the platform rules for reviewing merchant deals before they go live.
package moderation

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/grupon/grupon/services/api/internal/domain/catalog"
)

// Deal statuses used by the review workflow. They mirror the deal_status enum.
const (
	statusPendingReview = "pending_review"
	statusRejected      = "rejected"
	statusScheduled     = "scheduled"
	statusActive        = "active"
)

// Errors returned by moderation. The HTTP layer maps them to problem codes.
var (
	ErrNotFound     = errors.New("moderation: deal not found")
	ErrNotPending   = errors.New("moderation: deal is not waiting for review")
	ErrForbidden    = errors.New("moderation: role may not moderate deals")
	ErrReasonNeeded = errors.New("moderation: rejection reason is required")
)

// InvalidReasonError reports a rejection reason that is too short or too long.
type InvalidReasonError struct{ Code string }

func (e *InvalidReasonError) Error() string { return "moderation: invalid reason: " + e.Code }

// Actor is the staff member making the decision. Role is the highest moderation role they hold.
type Actor struct {
	UserID uuid.UUID
	Role   string
}

// Record is the review state of a deal.
type Record struct {
	ID         uuid.UUID
	MerchantID uuid.UUID
	Status     string
	StartsAt   time.Time
}

// AuditEntry records a moderation decision.
type AuditEntry struct {
	ActorID    uuid.UUID
	Role       string
	Action     string
	EntityType string
	EntityID   uuid.UUID
	After      map[string]any
}

// Repository persists moderation decisions. Transitions are conditional on the deal still being under review.
type Repository interface {
	ListPendingIDs(ctx context.Context, page catalog.Page) ([]uuid.UUID, int64, error)
	Summaries(ctx context.Context, ids []uuid.UUID) ([]catalog.DealSummary, error)
	Record(ctx context.Context, dealID uuid.UUID) (Record, error)
	Transition(ctx context.Context, dealID uuid.UUID, to, reason string, audit AuditEntry) error
	Detail(ctx context.Context, dealID uuid.UUID) (catalog.Deal, error)
}

// Moderators may review deals. Finance, support and merchant roles may not.
var moderatorRoles = map[string]bool{"superadmin": true, "admin": true, "moderator": true}

// Service applies the review decisions.
type Service struct {
	repo Repository
	now  func() time.Time
}

// NewService binds the moderation rules to a repository.
func NewService(repo Repository, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{repo: repo, now: now}
}

// Queue lists deals waiting for review, oldest update first.
func (s *Service) Queue(ctx context.Context, actor Actor, page catalog.Page) ([]catalog.DealSummary, int64, error) {
	if !moderatorRoles[actor.Role] {
		return nil, 0, ErrForbidden
	}
	ids, total, err := s.repo.ListPendingIDs(ctx, page)
	if err != nil {
		return nil, 0, err
	}
	items, err := s.repo.Summaries(ctx, ids)
	return items, total, err
}

// Preview returns the full deal exactly as a buyer would see it, for the reviewer.
func (s *Service) Preview(ctx context.Context, actor Actor, dealID uuid.UUID) (catalog.Deal, error) {
	if !moderatorRoles[actor.Role] {
		return catalog.Deal{}, ErrForbidden
	}
	return s.repo.Detail(ctx, dealID)
}

// Approve publishes a deal: scheduled if it starts in the future, active otherwise.
func (s *Service) Approve(ctx context.Context, actor Actor, dealID uuid.UUID) (catalog.Deal, error) {
	record, err := s.pending(ctx, actor, dealID)
	if err != nil {
		return catalog.Deal{}, err
	}
	to := statusActive
	if record.StartsAt.After(s.now()) {
		to = statusScheduled
	}
	if err := s.repo.Transition(ctx, dealID, to, "", s.audit(actor, dealID, "deal_approved", to)); err != nil {
		return catalog.Deal{}, err
	}
	return s.repo.Detail(ctx, dealID)
}

// Reject sends a deal back to the merchant with a reason the merchant can read.
func (s *Service) Reject(ctx context.Context, actor Actor, dealID uuid.UUID, reason string) (catalog.Deal, error) {
	reason = strings.TrimSpace(reason)
	if n := len([]rune(reason)); n < 3 || n > 500 {
		return catalog.Deal{}, &InvalidReasonError{Code: "length"}
	}
	if _, err := s.pending(ctx, actor, dealID); err != nil {
		return catalog.Deal{}, err
	}
	audit := s.audit(actor, dealID, "deal_rejected", statusRejected)
	audit.After["reason"] = reason
	if err := s.repo.Transition(ctx, dealID, statusRejected, reason, audit); err != nil {
		return catalog.Deal{}, err
	}
	return s.repo.Detail(ctx, dealID)
}

func (s *Service) pending(ctx context.Context, actor Actor, dealID uuid.UUID) (Record, error) {
	if !moderatorRoles[actor.Role] {
		return Record{}, ErrForbidden
	}
	record, err := s.repo.Record(ctx, dealID)
	if err != nil {
		return Record{}, err
	}
	if record.Status != statusPendingReview {
		return Record{}, ErrNotPending
	}
	return record, nil
}

func (s *Service) audit(actor Actor, dealID uuid.UUID, action, to string) AuditEntry {
	return AuditEntry{ActorID: actor.UserID, Role: actor.Role, Action: action, EntityType: "deal", EntityID: dealID, After: map[string]any{"status": to}}
}
