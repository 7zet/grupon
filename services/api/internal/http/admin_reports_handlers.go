package httpapi

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/grupon/grupon/services/api/internal/domain/catalog"
	"github.com/grupon/grupon/services/api/internal/domain/identity"
	"github.com/grupon/grupon/services/api/internal/domain/moderation"
	"github.com/grupon/grupon/services/api/internal/http/gen"
)

// AdminReportsAPI serves review reports, hiding reviews and the audit trail for platform staff.
type AdminReportsAPI struct {
	reports *moderation.ReportService
	auth    *AuthAPI
	logger  *slog.Logger
}

// NewAdminReportsAPI binds report decisions and the audit reader to the HTTP contract.
func NewAdminReportsAPI(service *moderation.ReportService, auth *AuthAPI, logger *slog.Logger) *AdminReportsAPI {
	return &AdminReportsAPI{reports: service, auth: auth, logger: logger}
}

// staff authenticates the caller and picks the highest staff role they hold.
func (a *AdminReportsAPI) staff(w http.ResponseWriter, r *http.Request) (moderation.Actor, bool) {
	account, ok := a.auth.authenticate(w, r)
	if !ok {
		return moderation.Actor{}, false
	}
	for _, role := range []identity.Role{identity.RoleSuperadmin, identity.RoleAdmin, identity.RoleModerator} {
		if account.HasRole(role) {
			return moderation.Actor{UserID: account.ID, Role: string(role)}, true
		}
	}
	writeProblem(w, r, http.StatusForbidden, "forbidden", nil, nil)
	return moderation.Actor{}, false
}

func (a *AdminReportsAPI) listReviewReports(w http.ResponseWriter, r *http.Request, page *int) {
	actor, ok := a.staff(w, r)
	if !ok {
		return
	}
	pageParams, err := catalog.NewPage(page, nil)
	if err != nil {
		writeProblem(w, r, http.StatusUnprocessableEntity, "invalid_page", nil, map[string][]string{"page": {"invalid_page"}})
		return
	}
	reports, total, err := a.reports.Open(r.Context(), actor, pageParams)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	items := make([]gen.ReviewReport, 0, len(reports))
	for _, report := range reports {
		items = append(items, gen.ReviewReport{
			Id:        gen.Uuid(report.ID),
			Reason:    report.Reason,
			Status:    gen.ReviewReportStatus(report.Status),
			CreatedAt: report.CreatedAt,
			Review:    mapReview(report.Review),
		})
	}
	writeJSON(w, http.StatusOK, gen.ReviewReportPage{Items: items, Page: pageParams.Number, PageSize: pageParams.Size, Total: total})
}

func (a *AdminReportsAPI) dismissReviewReport(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	actor, ok := a.staff(w, r)
	if !ok {
		return
	}
	if err := a.reports.Dismiss(r.Context(), actor, id); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *AdminReportsAPI) hideReview(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	actor, ok := a.staff(w, r)
	if !ok {
		return
	}
	if err := a.reports.Hide(r.Context(), actor, id); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *AdminReportsAPI) listAuditLogs(w http.ResponseWriter, r *http.Request, page *int) {
	actor, ok := a.staff(w, r)
	if !ok {
		return
	}
	pageParams, err := catalog.NewPage(page, nil)
	if err != nil {
		writeProblem(w, r, http.StatusUnprocessableEntity, "invalid_page", nil, map[string][]string{"page": {"invalid_page"}})
		return
	}
	entries, total, err := a.reports.Audit(r.Context(), actor, pageParams)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	items := make([]gen.AuditLog, 0, len(entries))
	for _, entry := range entries {
		item := gen.AuditLog{
			Id:         gen.Uuid(entry.ID),
			ActorRole:  entry.ActorRole,
			Action:     entry.Action,
			EntityType: entry.EntityType,
			CreatedAt:  entry.CreatedAt,
		}
		if entry.ActorUserID != uuid.Nil {
			actorID := gen.Uuid(entry.ActorUserID)
			item.ActorUserId = &actorID
		}
		if entry.EntityID != uuid.Nil {
			entityID := gen.Uuid(entry.EntityID)
			item.EntityId = &entityID
		}
		if entry.AfterData != nil {
			data := entry.AfterData
			item.AfterData = &data
		}
		items = append(items, item)
	}
	writeJSON(w, http.StatusOK, gen.AuditLogPage{Items: items, Page: pageParams.Number, PageSize: pageParams.Size, Total: total})
}

func (a *AdminReportsAPI) fail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, moderation.ErrForbidden), errors.Is(err, moderation.ErrAuditForbidden):
		writeProblem(w, r, http.StatusForbidden, "forbidden", nil, nil)
	case errors.Is(err, moderation.ErrNotFound):
		writeProblem(w, r, http.StatusNotFound, "not_found", nil, nil)
	case errors.Is(err, moderation.ErrNotOpen):
		writeProblem(w, r, http.StatusConflict, "report_not_open", nil, nil)
	default:
		a.logger.ErrorContext(r.Context(), "admin report request failed",
			slog.String("request_id", middleware.GetReqID(r.Context())),
			slog.String("error", err.Error()),
		)
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", nil, nil)
	}
}

// Delegation from the router.

func (s apiServer) ListReviewReports(w http.ResponseWriter, r *http.Request, params gen.ListReviewReportsParams) {
	if s.adminReports != nil {
		s.adminReports.listReviewReports(w, r, params.Page)
		return
	}
	s.ServerInterface.ListReviewReports(w, r, params)
}

func (s apiServer) DismissReviewReport(w http.ResponseWriter, r *http.Request, id gen.Id) {
	if s.adminReports != nil {
		s.adminReports.dismissReviewReport(w, r, uuid.UUID(id))
		return
	}
	s.ServerInterface.DismissReviewReport(w, r, id)
}

func (s apiServer) HideReview(w http.ResponseWriter, r *http.Request, id gen.Id) {
	if s.adminReports != nil {
		s.adminReports.hideReview(w, r, uuid.UUID(id))
		return
	}
	s.ServerInterface.HideReview(w, r, id)
}

func (s apiServer) ListAuditLogs(w http.ResponseWriter, r *http.Request, params gen.ListAuditLogsParams) {
	if s.adminReports != nil {
		s.adminReports.listAuditLogs(w, r, params.Page)
		return
	}
	s.ServerInterface.ListAuditLogs(w, r, params)
}

func (a *AdminReportsAPI) listFraudSignals(w http.ResponseWriter, r *http.Request, page *int) {
	actor, ok := a.staff(w, r)
	if !ok {
		return
	}
	pageParams, err := catalog.NewPage(page, nil)
	if err != nil {
		writeProblem(w, r, http.StatusUnprocessableEntity, "invalid_page", nil, map[string][]string{"page": {"invalid_page"}})
		return
	}
	signals, total, err := a.reports.FraudSignals(r.Context(), actor, pageParams)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	items := make([]gen.FraudSignal, 0, len(signals))
	for _, signal := range signals {
		item := gen.FraudSignal{
			Id:        gen.Uuid(signal.ID),
			Rule:      signal.Rule,
			Status:    gen.FraudSignalStatus(signal.Status),
			Detail:    signal.Detail,
			CreatedAt: signal.CreatedAt,
		}
		if signal.MerchantID != uuid.Nil {
			id := gen.Uuid(signal.MerchantID)
			item.MerchantId = &id
		}
		if signal.MerchantName != "" {
			name := signal.MerchantName
			item.MerchantName = &name
		}
		if signal.SubjectUserID != uuid.Nil {
			id := gen.Uuid(signal.SubjectUserID)
			item.SubjectUserId = &id
		}
		items = append(items, item)
	}
	writeJSON(w, http.StatusOK, gen.FraudSignalPage{Items: items, Page: pageParams.Number, PageSize: pageParams.Size, Total: total})
}

func (a *AdminReportsAPI) getOverview(w http.ResponseWriter, r *http.Request) {
	actor, ok := a.staff(w, r)
	if !ok {
		return
	}
	overview, err := a.reports.Overview(r.Context(), actor)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, gen.AdminOverview{Users: overview.Users, Businesses: overview.Businesses, ActiveDeals: overview.ActiveDeals, OrdersToday: overview.OrdersToday})
}
