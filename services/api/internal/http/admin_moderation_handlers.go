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

// ModerationAPI serves the review queue. Only platform staff with a moderation role may call it.
type ModerationAPI struct {
	moderation *moderation.Service
	auth       *AuthAPI
	logger     *slog.Logger
}

// NewModerationAPI binds the moderation rules to the HTTP contract.
func NewModerationAPI(service *moderation.Service, auth *AuthAPI, logger *slog.Logger) *ModerationAPI {
	return &ModerationAPI{moderation: service, auth: auth, logger: logger}
}

// actor authenticates the caller and picks the highest moderation role they hold.
func (a *ModerationAPI) actor(w http.ResponseWriter, r *http.Request) (moderation.Actor, bool) {
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

func (a *ModerationAPI) listModerationDeals(w http.ResponseWriter, r *http.Request, page *int) {
	actor, ok := a.actor(w, r)
	if !ok {
		return
	}
	pageParams, err := catalog.NewPage(page, nil)
	if err != nil {
		writeProblem(w, r, http.StatusUnprocessableEntity, "invalid_page", nil, map[string][]string{"page": {"invalid_page"}})
		return
	}
	items, total, err := a.moderation.Queue(r.Context(), actor, pageParams)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	cards := make([]gen.DealSummary, 0, len(items))
	for _, item := range items {
		cards = append(cards, mapDealSummary(item))
	}
	writeJSON(w, http.StatusOK, gen.DealPage{Items: cards, Page: pageParams.Number, PageSize: pageParams.Size, Total: total})
}

func (a *ModerationAPI) approveDeal(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	actor, ok := a.actor(w, r)
	if !ok {
		return
	}
	deal, err := a.moderation.Approve(r.Context(), actor, id)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapDeal(deal))
}

func (a *ModerationAPI) rejectDeal(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	actor, ok := a.actor(w, r)
	if !ok {
		return
	}
	var body gen.RejectDealInput
	if _, ok := decodeBody(w, r, &body); !ok {
		return
	}
	deal, err := a.moderation.Reject(r.Context(), actor, id, body.Reason)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapDeal(deal))
}

func (a *ModerationAPI) fail(w http.ResponseWriter, r *http.Request, err error) {
	var reason *moderation.InvalidReasonError
	switch {
	case errors.As(err, &reason):
		writeProblem(w, r, http.StatusUnprocessableEntity, "invalid_reason", nil, map[string][]string{"reason": {reason.Code}})
	case errors.Is(err, moderation.ErrForbidden):
		writeProblem(w, r, http.StatusForbidden, "forbidden", nil, nil)
	case errors.Is(err, moderation.ErrNotFound):
		writeProblem(w, r, http.StatusNotFound, "not_found", nil, nil)
	case errors.Is(err, moderation.ErrNotPending):
		writeProblem(w, r, http.StatusConflict, "deal_not_pending_review", nil, nil)
	case errors.Is(err, moderation.ErrCategoryPublishDisabled):
		writeProblem(w, r, http.StatusUnprocessableEntity, "category_publish_disabled", nil, nil)
	default:
		a.logger.ErrorContext(r.Context(), "moderation request failed",
			slog.String("request_id", middleware.GetReqID(r.Context())),
			slog.String("error", err.Error()),
		)
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", nil, nil)
	}
}

// Delegation from the router. Each operation is served here when the moderation API is mounted.

func (s apiServer) ListModerationDeals(w http.ResponseWriter, r *http.Request, params gen.ListModerationDealsParams) {
	if s.moderation != nil {
		s.moderation.listModerationDeals(w, r, params.Page)
		return
	}
	s.ServerInterface.ListModerationDeals(w, r, params)
}

func (s apiServer) ApproveDeal(w http.ResponseWriter, r *http.Request, id gen.Id) {
	if s.moderation != nil {
		s.moderation.approveDeal(w, r, uuid.UUID(id))
		return
	}
	s.ServerInterface.ApproveDeal(w, r, id)
}

func (s apiServer) RejectDeal(w http.ResponseWriter, r *http.Request, id gen.Id) {
	if s.moderation != nil {
		s.moderation.rejectDeal(w, r, uuid.UUID(id))
		return
	}
	s.ServerInterface.RejectDeal(w, r, id)
}
