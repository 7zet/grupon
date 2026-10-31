package moderation

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/grupon/grupon/services/api/internal/domain/catalog"
	"github.com/grupon/grupon/services/api/internal/domain/reviews"
)

// ErrNotOpen reports a report that is already closed or does not exist.
var ErrNotOpen = errors.New("moderation: report is not open")

// ErrAuditForbidden reports a role that may not read the audit trail.
var ErrAuditForbidden = errors.New("moderation: role may not read the audit log")

// auditRoles may read the audit trail. Moderators cannot.
var auditRoles = map[string]bool{"superadmin": true, "admin": true}

// Report is a complaint about a review.
type Report struct {
	ID        uuid.UUID
	Reason    string
	Status    string
	CreatedAt time.Time
	Review    reviews.Review
}

// AuditRow is one entry of the audit trail.
type AuditRow struct {
	ID          uuid.UUID
	ActorUserID uuid.UUID
	ActorRole   string
	Action      string
	EntityType  string
	EntityID    uuid.UUID
	AfterData   map[string]any
	CreatedAt   time.Time
}

// ReportRepository reads and closes review reports and reads the audit trail.
type ReportRepository interface {
	OpenReports(ctx context.Context, page catalog.Page) ([]Report, int64, error)
	Dismiss(ctx context.Context, reportID uuid.UUID, audit AuditEntry) (bool, error)
	// HideReview hides the review, closes its open reports and writes the audit row in one transaction.
	HideReview(ctx context.Context, reviewID uuid.UUID, audit AuditEntry) (bool, error)
	AuditRows(ctx context.Context, page catalog.Page) ([]AuditRow, int64, error)
}

// ReportService applies moderator decisions on reviews.
type ReportService struct {
	repo ReportRepository
}

// NewReportService binds report decisions to a repository.
func NewReportService(repo ReportRepository) *ReportService {
	return &ReportService{repo: repo}
}

// Open lists open reports, oldest first.
func (s *ReportService) Open(ctx context.Context, actor Actor, page catalog.Page) ([]Report, int64, error) {
	if !moderatorRoles[actor.Role] {
		return nil, 0, ErrForbidden
	}
	return s.repo.OpenReports(ctx, page)
}

// Dismiss closes a report and leaves the review as it is.
func (s *ReportService) Dismiss(ctx context.Context, actor Actor, reportID uuid.UUID) error {
	if !moderatorRoles[actor.Role] {
		return ErrForbidden
	}
	closed, err := s.repo.Dismiss(ctx, reportID, AuditEntry{
		ActorID: actor.UserID, Role: actor.Role, Action: "review_report_dismissed", EntityType: "review_report", EntityID: reportID,
		After: map[string]any{"status": "dismissed"},
	})
	if err != nil {
		return err
	}
	if !closed {
		return ErrNotOpen
	}
	return nil
}

// Hide removes a review from public pages and closes every open report on it.
func (s *ReportService) Hide(ctx context.Context, actor Actor, reviewID uuid.UUID) error {
	if !moderatorRoles[actor.Role] {
		return ErrForbidden
	}
	hidden, err := s.repo.HideReview(ctx, reviewID, AuditEntry{
		ActorID: actor.UserID, Role: actor.Role, Action: "review_hidden", EntityType: "review", EntityID: reviewID,
		After: map[string]any{"status": reviews.StatusHidden},
	})
	if err != nil {
		return err
	}
	if !hidden {
		return ErrNotFound
	}
	return nil
}

// Audit returns one page of the audit trail, newest first. Only admins and superadmins may read it.
func (s *ReportService) Audit(ctx context.Context, actor Actor, page catalog.Page) ([]AuditRow, int64, error) {
	if !auditRoles[actor.Role] {
		return nil, 0, ErrAuditForbidden
	}
	return s.repo.AuditRows(ctx, page)
}
