package moderation

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/grupon/grupon/services/api/internal/domain/catalog"
)

type fakeReports struct {
	open    map[uuid.UUID]bool
	hidden  map[uuid.UUID]bool
	audited []string
}

func (f *fakeReports) OpenReports(context.Context, catalog.Page) ([]Report, int64, error) {
	return nil, 0, nil
}

func (f *fakeReports) Dismiss(_ context.Context, id uuid.UUID, audit AuditEntry) (bool, error) {
	if !f.open[id] {
		return false, nil
	}
	delete(f.open, id)
	f.audited = append(f.audited, audit.Action)
	return true, nil
}

func (f *fakeReports) HideReview(_ context.Context, id uuid.UUID, audit AuditEntry) (bool, error) {
	f.hidden[id] = true
	f.audited = append(f.audited, audit.Action)
	return true, nil
}

func (f *fakeReports) AuditRows(context.Context, catalog.Page) ([]AuditRow, int64, error) {
	return nil, 0, nil
}

func (f *fakeReports) FraudSignals(context.Context, catalog.Page) ([]FraudSignal, int64, error) {
	return nil, 0, nil
}

func (f *fakeReports) Overview(context.Context) (Overview, error) { return Overview{Users: 3}, nil }

func TestFraudSignalsAreForAdminsAndOverviewForAllStaff(t *testing.T) {
	service := NewReportService(&fakeReports{})
	ctx := context.Background()
	if _, _, err := service.FraudSignals(ctx, Actor{Role: "moderator"}, catalog.Page{}); !errors.Is(err, ErrAuditForbidden) {
		t.Fatalf("moderator fraud signals: got %v", err)
	}
	if _, _, err := service.FraudSignals(ctx, Actor{Role: "admin"}, catalog.Page{}); err != nil {
		t.Fatal(err)
	}
	if got, err := service.Overview(ctx, Actor{Role: "moderator"}); err != nil || got.Users != 3 {
		t.Fatalf("moderator overview: %v %v", got, err)
	}
	if _, err := service.Overview(ctx, Actor{Role: "customer"}); !errors.Is(err, ErrAuditForbidden) {
		t.Fatalf("customer overview: got %v", err)
	}
}

func TestDismissClosesOnlyOpenReportsAndAudits(t *testing.T) {
	report := uuid.New()
	repo := &fakeReports{open: map[uuid.UUID]bool{report: true}, hidden: map[uuid.UUID]bool{}}
	service := NewReportService(repo)
	moderator := Actor{UserID: uuid.New(), Role: "moderator"}

	if err := service.Dismiss(context.Background(), moderator, report); err != nil {
		t.Fatal(err)
	}
	if err := service.Dismiss(context.Background(), moderator, report); !errors.Is(err, ErrNotOpen) {
		t.Fatalf("second dismiss: got %v", err)
	}
	if len(repo.audited) != 1 || repo.audited[0] != "review_report_dismissed" {
		t.Fatalf("audit=%v", repo.audited)
	}
}

func TestHideIsAuditedAndStaffOnly(t *testing.T) {
	review := uuid.New()
	repo := &fakeReports{open: map[uuid.UUID]bool{}, hidden: map[uuid.UUID]bool{}}
	service := NewReportService(repo)

	if err := service.Hide(context.Background(), Actor{UserID: uuid.New(), Role: "finance"}, review); !errors.Is(err, ErrForbidden) {
		t.Fatalf("finance hide: got %v", err)
	}
	if err := service.Hide(context.Background(), Actor{UserID: uuid.New(), Role: "moderator"}, review); err != nil {
		t.Fatal(err)
	}
	if !repo.hidden[review] || repo.audited[0] != "review_hidden" {
		t.Fatalf("hidden=%v audit=%v", repo.hidden[review], repo.audited)
	}
}

func TestAuditTrailIsNotForModerators(t *testing.T) {
	service := NewReportService(&fakeReports{})
	if _, _, err := service.Audit(context.Background(), Actor{UserID: uuid.New(), Role: "moderator"}, catalog.Page{Number: 1, Size: 20}); !errors.Is(err, ErrAuditForbidden) {
		t.Fatalf("moderator audit: got %v", err)
	}
	if _, _, err := service.Audit(context.Background(), Actor{UserID: uuid.New(), Role: "admin"}, catalog.Page{Number: 1, Size: 20}); err != nil {
		t.Fatalf("admin audit: %v", err)
	}
}
