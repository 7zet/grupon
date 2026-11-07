package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/grupon/grupon/services/api/internal/domain/identity"
	"github.com/grupon/grupon/services/api/internal/domain/merchant"
	"github.com/grupon/grupon/services/api/internal/http/gen"
)

// MerchantReviewAPI serves the platform queue for businesses (FR-01.3). The rules live in the merchant
// domain; this layer only maps the caller and the errors.
type MerchantReviewAPI struct {
	service *merchant.ReviewService
	auth    *AuthAPI
	logger  *slog.Logger
}

// NewMerchantReviewAPI binds the review rules to the HTTP contract.
func NewMerchantReviewAPI(service *merchant.ReviewService, auth *AuthAPI, logger *slog.Logger) *MerchantReviewAPI {
	return &MerchantReviewAPI{service: service, auth: auth, logger: logger}
}

// actor authenticates the caller and picks the highest review role they hold.
func (a *MerchantReviewAPI) actor(w http.ResponseWriter, r *http.Request) (merchant.ReviewActor, bool) {
	account, ok := a.auth.authenticate(w, r)
	if !ok {
		return merchant.ReviewActor{}, false
	}
	for _, role := range []identity.Role{identity.RoleSuperadmin, identity.RoleAdmin, identity.RoleModerator} {
		if account.HasRole(role) {
			return merchant.ReviewActor{UserID: account.ID, Role: string(role)}, true
		}
	}
	writeProblem(w, r, http.StatusForbidden, "forbidden", nil, nil)
	return merchant.ReviewActor{}, false
}

func (a *MerchantReviewAPI) ListMerchantReviewQueue(w http.ResponseWriter, r *http.Request, params gen.ListMerchantReviewQueueParams) {
	actor, ok := a.actor(w, r)
	if !ok {
		return
	}
	status := ""
	if params.Status != nil {
		status = string(*params.Status)
	}
	items, err := a.service.List(r.Context(), actor, status)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	out := make([]gen.MerchantReview, 0, len(items))
	for _, item := range items {
		out = append(out, gen.MerchantReview{
			Id:              gen.Uuid(item.ID),
			LegalName:       item.LegalName,
			DisplayName:     item.DisplayName,
			Stir:            item.STIR,
			OrgForm:         gen.MerchantReviewOrgForm(item.OrgForm),
			TaxRegime:       gen.MerchantReviewTaxRegime(item.TaxRegime),
			BankMfo:         item.BankMFO,
			BankAccount:     item.BankAccount,
			ResponsibleName: item.ResponsibleName,
			ReviewStatus:    gen.MerchantReviewReviewStatus(item.ReviewStatus),
			SubmittedAt:     item.SubmittedAt,
			Documents:       documentKinds(item.Documents),
		})
	}
	writeJSON(w, http.StatusOK, gen.MerchantReviewList{Items: out})
}

func (a *MerchantReviewAPI) StartMerchantReview(w http.ResponseWriter, r *http.Request, id gen.Id) {
	actor, ok := a.actor(w, r)
	if !ok {
		return
	}
	if err := a.service.Start(r.Context(), actor, uuid.UUID(id)); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *MerchantReviewAPI) ApproveMerchant(w http.ResponseWriter, r *http.Request, id gen.Id) {
	actor, ok := a.actor(w, r)
	if !ok {
		return
	}
	if err := a.service.Approve(r.Context(), actor, uuid.UUID(id)); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *MerchantReviewAPI) RejectMerchant(w http.ResponseWriter, r *http.Request, id gen.Id) {
	actor, ok := a.actor(w, r)
	if !ok {
		return
	}
	var body gen.RejectMerchantRequest
	if _, ok := decodeBody(w, r, &body); !ok {
		return
	}
	if err := a.service.Reject(r.Context(), actor, uuid.UUID(id), body.Reason); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *MerchantReviewAPI) SuspendMerchant(w http.ResponseWriter, r *http.Request, id gen.Id) {
	a.withReason(w, r, id, a.service.Suspend)
}

func (a *MerchantReviewAPI) TerminateMerchant(w http.ResponseWriter, r *http.Request, id gen.Id) {
	a.withReason(w, r, id, a.service.Terminate)
}

func (a *MerchantReviewAPI) ReinstateMerchant(w http.ResponseWriter, r *http.Request, id gen.Id) {
	actor, ok := a.actor(w, r)
	if !ok {
		return
	}
	if err := a.service.Reinstate(r.Context(), actor, uuid.UUID(id)); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *MerchantReviewAPI) withReason(w http.ResponseWriter, r *http.Request, id gen.Id, decide func(context.Context, merchant.ReviewActor, uuid.UUID, string) error) {
	actor, ok := a.actor(w, r)
	if !ok {
		return
	}
	var body gen.RejectMerchantRequest
	if _, ok := decodeBody(w, r, &body); !ok {
		return
	}
	if err := decide(r.Context(), actor, uuid.UUID(id), body.Reason); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *MerchantReviewAPI) fail(w http.ResponseWriter, r *http.Request, err error) {
	var invalid *merchant.InvalidInputError
	switch {
	case errors.As(err, &invalid):
		writeProblem(w, r, http.StatusUnprocessableEntity, invalid.Code, nil,
			map[string][]string{invalid.Field: {invalid.Code}})
	case errors.Is(err, merchant.ErrForbidden):
		writeProblem(w, r, http.StatusForbidden, "forbidden", nil, nil)
	case errors.Is(err, merchant.ErrBusinessNotFound):
		writeProblem(w, r, http.StatusNotFound, "business_not_found", nil, nil)
	case errors.Is(err, merchant.ErrInvalidTransition):
		writeProblem(w, r, http.StatusConflict, "invalid_transition", nil, nil)
	default:
		a.logger.ErrorContext(r.Context(), "merchant review failed",
			slog.String("request_id", middleware.GetReqID(r.Context())),
			slog.String("error", err.Error()),
		)
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", nil, nil)
	}
}

func documentKinds(kinds []string) *[]gen.MerchantReviewDocuments {
	out := make([]gen.MerchantReviewDocuments, 0, len(kinds))
	for _, kind := range kinds {
		out = append(out, gen.MerchantReviewDocuments(kind))
	}
	return &out
}
