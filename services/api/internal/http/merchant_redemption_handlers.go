package httpapi

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/grupon/grupon/services/api/internal/domain/merchant"
	"github.com/grupon/grupon/services/api/internal/domain/redemption"
	"github.com/grupon/grupon/services/api/internal/http/gen"
)

// MerchantRedemptionAPI serves voucher lookup and redemption at the counter. Cashiers may use it.
type MerchantRedemptionAPI struct {
	redemption *redemption.Service
	workspace  *merchant.WorkspaceService
	auth       *AuthAPI
	logger     *slog.Logger
}

// NewMerchantRedemptionAPI binds the redemption rules to the HTTP contract.
func NewMerchantRedemptionAPI(service *redemption.Service, workspace *merchant.WorkspaceService, auth *AuthAPI, logger *slog.Logger) *MerchantRedemptionAPI {
	return &MerchantRedemptionAPI{redemption: service, workspace: workspace, auth: auth, logger: logger}
}

func (a *MerchantRedemptionAPI) LookupMerchantVoucher(w http.ResponseWriter, r *http.Request) {
	actor, merchantID, ok := a.scope(w, r)
	if !ok {
		return
	}
	var body gen.VoucherLookupRequest
	if _, ok := decodeBody(w, r, &body); !ok {
		return
	}
	token, code := "", ""
	if body.QrToken != nil {
		token = *body.QrToken
	}
	if body.Code != nil {
		code = *body.Code
	}
	view, err := a.redemption.Lookup(r.Context(), merchantID, actor.UserID, token, code)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapVoucher(view))
}

func (a *MerchantRedemptionAPI) RedeemMerchantVoucher(w http.ResponseWriter, r *http.Request, id gen.Id) {
	actor, merchantID, ok := a.scope(w, r)
	if !ok {
		return
	}
	var body gen.RedeemRequest
	if _, ok := decodeBody(w, r, &body); !ok {
		return
	}
	var geo *redemption.Geo
	if body.Lat != nil && body.Lng != nil {
		geo = &redemption.Geo{Lat: float64(*body.Lat), Lng: float64(*body.Lng)}
	}
	view, err := a.redemption.Redeem(r.Context(), merchantID, actor, uuid.UUID(id), uuid.UUID(body.LocationId), geo)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapVoucher(view))
}

// scope authenticates the caller and resolves their merchant. Any merchant role may redeem.
func (a *MerchantRedemptionAPI) scope(w http.ResponseWriter, r *http.Request) (merchant.Actor, uuid.UUID, bool) {
	account, ok := a.auth.authenticate(w, r)
	if !ok {
		return merchant.Actor{}, uuid.Nil, false
	}
	merchantID, role, err := a.workspace.Authorize(r.Context(), account.ID, false)
	if err != nil {
		a.fail(w, r, err)
		return merchant.Actor{}, uuid.Nil, false
	}
	return merchant.Actor{UserID: account.ID, Role: role}, merchantID, true
}

func (a *MerchantRedemptionAPI) fail(w http.ResponseWriter, r *http.Request, err error) {
	var invalid *merchant.InvalidInputError
	switch {
	case errors.As(err, &invalid):
		writeProblem(w, r, http.StatusUnprocessableEntity, invalid.Code, nil,
			map[string][]string{invalid.Field: {invalid.Code}})
	case errors.Is(err, redemption.ErrTooManyAttempts):
		writeProblem(w, r, http.StatusTooManyRequests, "too_many_attempts", nil, nil)
	case errors.Is(err, redemption.ErrScannerBlocked):
		writeProblem(w, r, http.StatusTooManyRequests, "scanner_blocked", nil, nil)
	case errors.Is(err, redemption.ErrInvalidToken):
		writeProblem(w, r, http.StatusUnprocessableEntity, "invalid_qr", nil, nil)
	case errors.Is(err, redemption.ErrWrongMerchant), errors.Is(err, merchant.ErrNoMerchant), errors.Is(err, merchant.ErrForbidden):
		writeProblem(w, r, http.StatusForbidden, "wrong_merchant", nil, nil)
	case errors.Is(err, redemption.ErrNotFound):
		writeProblem(w, r, http.StatusNotFound, "voucher_not_found", nil, nil)
	case errors.Is(err, redemption.ErrAlreadyRedeemed):
		writeProblem(w, r, http.StatusConflict, "already_redeemed", nil, nil)
	case errors.Is(err, redemption.ErrExpired):
		writeProblem(w, r, http.StatusUnprocessableEntity, "voucher_expired", nil, nil)
	case errors.Is(err, redemption.ErrRefunded):
		writeProblem(w, r, http.StatusUnprocessableEntity, "voucher_refunded", nil, nil)
	case errors.Is(err, redemption.ErrOnHold):
		writeProblem(w, r, http.StatusConflict, "voucher_on_hold", nil, nil)
	case errors.Is(err, redemption.ErrCategoryNotRedeemable):
		writeProblem(w, r, http.StatusUnprocessableEntity, "category_not_redeemable", nil, nil)
	case errors.Is(err, redemption.ErrVoided):
		writeProblem(w, r, http.StatusUnprocessableEntity, "voucher_voided", nil, nil)
	default:
		a.logger.ErrorContext(r.Context(), "redemption request failed",
			slog.String("request_id", middleware.GetReqID(r.Context())),
			slog.String("error", err.Error()),
		)
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", nil, nil)
	}
}

// Delegation from the router.

func (s apiServer) LookupMerchantVoucher(w http.ResponseWriter, r *http.Request) {
	if s.redeem != nil {
		s.redeem.LookupMerchantVoucher(w, r)
		return
	}
	s.ServerInterface.LookupMerchantVoucher(w, r)
}

func (s apiServer) RedeemMerchantVoucher(w http.ResponseWriter, r *http.Request, id gen.Id) {
	if s.redeem != nil {
		s.redeem.RedeemMerchantVoucher(w, r, id)
		return
	}
	s.ServerInterface.RedeemMerchantVoucher(w, r, id)
}
