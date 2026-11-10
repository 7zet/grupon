package httpapi

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/dealo/dealo/services/api/internal/domain/merchant"
	"github.com/dealo/dealo/services/api/internal/http/gen"
)

// MerchantStaffAPI manages the people who work in a business (PRD FR-01.6). The owner decides; everyone
// in the business may read the list.
type MerchantStaffAPI struct {
	service   *merchant.StaffService
	workspace *merchant.WorkspaceService
	auth      *AuthAPI
	logger    *slog.Logger
}

// NewMerchantStaffAPI binds the staff rules to the HTTP contract.
func NewMerchantStaffAPI(service *merchant.StaffService, workspace *merchant.WorkspaceService, auth *AuthAPI, logger *slog.Logger) *MerchantStaffAPI {
	return &MerchantStaffAPI{service: service, workspace: workspace, auth: auth, logger: logger}
}

func (a *MerchantStaffAPI) scope(w http.ResponseWriter, r *http.Request, mayEdit bool) (merchant.Actor, uuid.UUID, bool) {
	account, ok := a.auth.authenticate(w, r)
	if !ok {
		return merchant.Actor{}, uuid.Nil, false
	}
	merchantID, role, err := a.workspace.Authorize(r.Context(), account.ID, mayEdit)
	if err != nil {
		a.fail(w, r, err)
		return merchant.Actor{}, uuid.Nil, false
	}
	return merchant.Actor{UserID: account.ID, Role: role}, merchantID, true
}

func (a *MerchantStaffAPI) ListMerchantStaff(w http.ResponseWriter, r *http.Request) {
	_, merchantID, ok := a.scope(w, r, false)
	if !ok {
		return
	}
	members, err := a.service.List(r.Context(), merchantID)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	items := make([]gen.StaffMember, 0, len(members))
	for _, m := range members {
		items = append(items, mapStaffMember(m))
	}
	writeJSON(w, http.StatusOK, gen.StaffList{Items: items})
}

func (a *MerchantStaffAPI) InviteMerchantStaff(w http.ResponseWriter, r *http.Request) {
	actor, merchantID, ok := a.scope(w, r, true)
	if !ok {
		return
	}
	var body gen.StaffInput
	if _, ok := decodeBody(w, r, &body); !ok {
		return
	}
	member, err := a.service.Invite(r.Context(), actor, merchantID, body.Phone, string(body.Role))
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, mapStaffMember(member))
}

func (a *MerchantStaffAPI) UpdateMerchantStaff(w http.ResponseWriter, r *http.Request, id gen.Id) {
	actor, merchantID, ok := a.scope(w, r, true)
	if !ok {
		return
	}
	var body gen.StaffInput
	if _, ok := decodeBody(w, r, &body); !ok {
		return
	}
	member, err := a.service.ChangeRole(r.Context(), actor, merchantID, uuid.UUID(id), string(body.Role))
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapStaffMember(member))
}

func (a *MerchantStaffAPI) RemoveMerchantStaff(w http.ResponseWriter, r *http.Request, id gen.Id) {
	actor, merchantID, ok := a.scope(w, r, true)
	if !ok {
		return
	}
	if err := a.service.Remove(r.Context(), actor, merchantID, uuid.UUID(id)); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *MerchantStaffAPI) fail(w http.ResponseWriter, r *http.Request, err error) {
	var invalid *merchant.InvalidInputError
	switch {
	case errors.As(err, &invalid):
		writeProblem(w, r, http.StatusUnprocessableEntity, invalid.Code, nil,
			map[string][]string{invalid.Field: {invalid.Code}})
	case errors.Is(err, merchant.ErrUserNotFound):
		writeProblem(w, r, http.StatusNotFound, "user_not_found", nil, nil)
	case errors.Is(err, merchant.ErrStaffNotFound):
		writeProblem(w, r, http.StatusNotFound, "staff_not_found", nil, nil)
	case errors.Is(err, merchant.ErrAlreadyStaff):
		writeProblem(w, r, http.StatusConflict, "already_staff", nil, nil)
	case errors.Is(err, merchant.ErrOwnerProtected):
		writeProblem(w, r, http.StatusConflict, "owner_protected", nil, nil)
	case errors.Is(err, merchant.ErrForbidden):
		writeProblem(w, r, http.StatusForbidden, "forbidden", nil, nil)
	default:
		a.logger.ErrorContext(r.Context(), "merchant staff failed",
			slog.String("request_id", middleware.GetReqID(r.Context())),
			slog.String("error", err.Error()),
		)
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", nil, nil)
	}
}

func mapStaffMember(m merchant.StaffMember) gen.StaffMember {
	phone := m.Phone
	return gen.StaffMember{
		Id:   gen.Uuid(m.AssignmentID),
		Role: gen.StaffMemberRole(m.Role),
		User: gen.User{
			Id:    gen.Uuid(m.UserID),
			Email: openapi_types.Email(m.Email),
			Name:  m.Name,
			Phone: &phone,
		},
	}
}
