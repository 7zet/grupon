package httpapi

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/dealo/dealo/services/api/internal/domain/finance"
	"github.com/dealo/dealo/services/api/internal/domain/identity"
	"github.com/dealo/dealo/services/api/internal/domain/merchant"
	"github.com/dealo/dealo/services/api/internal/domain/redemption"
	"github.com/dealo/dealo/services/api/internal/http/gen"
)

// FinanceAPI serves reversals, refunds, disputes and payouts. Every write is audited or ledgered
// inside the domain; this layer only maps callers and errors.
type FinanceAPI struct {
	finance    *finance.Service
	redemption *redemption.Service
	workspace  *merchant.WorkspaceService
	auth       *AuthAPI
	logger     *slog.Logger
}

// NewFinanceAPI binds the finance and redemption rules to the HTTP contract.
func NewFinanceAPI(financeService *finance.Service, redemptionService *redemption.Service, workspace *merchant.WorkspaceService, auth *AuthAPI, logger *slog.Logger) *FinanceAPI {
	return &FinanceAPI{finance: financeService, redemption: redemptionService, workspace: workspace, auth: auth, logger: logger}
}

// staffActor returns the highest finance-related role the account holds.
func (a *FinanceAPI) staffActor(w http.ResponseWriter, r *http.Request) (finance.Actor, identity.Account, bool) {
	account, ok := a.auth.authenticate(w, r)
	if !ok {
		return finance.Actor{}, identity.Account{}, false
	}
	role := ""
	switch {
	case account.HasRole(identity.RoleSuperadmin):
		role = string(identity.RoleSuperadmin)
	case account.HasRole(identity.RoleFinance):
		role = string(identity.RoleFinance)
	case account.HasRole(identity.RoleSupport):
		role = string(identity.RoleSupport)
	}
	return finance.Actor{UserID: account.ID, Role: role}, account, true
}

func (a *FinanceAPI) ReverseMerchantVoucher(w http.ResponseWriter, r *http.Request, id gen.Id) {
	account, ok := a.auth.authenticate(w, r)
	if !ok {
		return
	}
	merchantID, role, err := a.workspace.Authorize(r.Context(), account.ID, false)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	var body gen.ReverseRequest
	if _, ok := decodeBody(w, r, &body); !ok {
		return
	}
	actor := merchant.Actor{UserID: account.ID, Role: role}
	if err := a.redemption.Reverse(r.Context(), merchantID, actor, uuid.UUID(id), body.Reason); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *FinanceAPI) RequestVoucherRefund(w http.ResponseWriter, r *http.Request, id gen.Id) {
	account, ok := a.auth.authenticate(w, r)
	if !ok {
		return
	}
	var body gen.ReverseRequest
	if _, ok := decodeBody(w, r, &body); !ok {
		return
	}
	if _, err := a.finance.RequestRefund(r.Context(), account.ID, uuid.UUID(id), body.Reason); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (a *FinanceAPI) OpenVoucherDispute(w http.ResponseWriter, r *http.Request, id gen.Id) {
	account, ok := a.auth.authenticate(w, r)
	if !ok {
		return
	}
	var body gen.ReverseRequest
	if _, ok := decodeBody(w, r, &body); !ok {
		return
	}
	if err := a.finance.OpenDispute(r.Context(), account.ID, uuid.UUID(id), body.Reason); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (a *FinanceAPI) PreparePayout(w http.ResponseWriter, r *http.Request, id gen.Id) {
	actor, _, ok := a.staffActor(w, r)
	if !ok {
		return
	}
	if _, err := a.finance.PreparePayout(r.Context(), actor, uuid.UUID(id)); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (a *FinanceAPI) ApprovePayout(w http.ResponseWriter, r *http.Request, id gen.Id) {
	actor, _, ok := a.staffActor(w, r)
	if !ok {
		return
	}
	var body gen.ApprovePayoutRequest
	if _, ok := decodeBody(w, r, &body); !ok {
		return
	}
	docRef := ""
	if body.BankDocRef != nil {
		docRef = *body.BankDocRef
	}
	if err := a.finance.ApprovePayout(r.Context(), actor, uuid.UUID(id), docRef); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *FinanceAPI) MarkRefundPaid(w http.ResponseWriter, r *http.Request, id gen.Id) {
	actor, _, ok := a.staffActor(w, r)
	if !ok {
		return
	}
	if err := a.finance.MarkRefundPaid(r.Context(), actor, uuid.UUID(id)); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *FinanceAPI) ReconcilePayout(w http.ResponseWriter, r *http.Request, id gen.Id) {
	actor, _, ok := a.staffActor(w, r)
	if !ok {
		return
	}
	if err := a.finance.MarkPayoutReconciled(r.Context(), actor, uuid.UUID(id)); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *FinanceAPI) ApproveRefund(w http.ResponseWriter, r *http.Request, id gen.Id) {
	actor, _, ok := a.staffActor(w, r)
	if !ok {
		return
	}
	if err := a.finance.ApproveRefund(r.Context(), actor, uuid.UUID(id)); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *FinanceAPI) ProposeRefund(w http.ResponseWriter, r *http.Request, id gen.Id) {
	actor, _, ok := a.staffActor(w, r)
	if !ok {
		return
	}
	if err := a.finance.ProposeRefund(r.Context(), actor, uuid.UUID(id)); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *FinanceAPI) ResolveDispute(w http.ResponseWriter, r *http.Request, id gen.Id) {
	actor, _, ok := a.staffActor(w, r)
	if !ok {
		return
	}
	var body gen.ResolveDisputeRequest
	if _, ok := decodeBody(w, r, &body); !ok {
		return
	}
	if err := a.finance.ResolveDispute(r.Context(), actor, uuid.UUID(id), string(body.Resolution)); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *FinanceAPI) RequestDisputeEvidence(w http.ResponseWriter, r *http.Request, id gen.Id) {
	actor, _, ok := a.staffActor(w, r)
	if !ok {
		return
	}
	if err := a.finance.RequestDisputeEvidence(r.Context(), actor, uuid.UUID(id)); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *FinanceAPI) StartDisputeReview(w http.ResponseWriter, r *http.Request, id gen.Id) {
	actor, _, ok := a.staffActor(w, r)
	if !ok {
		return
	}
	if err := a.finance.StartDisputeReview(r.Context(), actor, uuid.UUID(id)); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *FinanceAPI) CloseDispute(w http.ResponseWriter, r *http.Request, id gen.Id) {
	actor, _, ok := a.staffActor(w, r)
	if !ok {
		return
	}
	if err := a.finance.CloseDispute(r.Context(), actor, uuid.UUID(id)); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *FinanceAPI) fail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, redemption.ErrReversalWindowClosed):
		writeProblem(w, r, http.StatusConflict, "reversal_window_closed", nil, nil)
	case errors.Is(err, redemption.ErrNoRedemption), errors.Is(err, finance.ErrNoRedemption):
		writeProblem(w, r, http.StatusConflict, "no_redemption", nil, nil)
	case errors.Is(err, redemption.ErrWrongMerchant), errors.Is(err, merchant.ErrNoMerchant):
		writeProblem(w, r, http.StatusForbidden, "wrong_merchant", nil, nil)
	case errors.Is(err, merchant.ErrForbidden), errors.Is(err, finance.ErrForbidden):
		writeProblem(w, r, http.StatusForbidden, "forbidden", nil, nil)
	case errors.Is(err, finance.ErrNotOwner):
		writeProblem(w, r, http.StatusNotFound, "not_found", nil, nil)
	case errors.Is(err, finance.ErrNotFound), errors.Is(err, redemption.ErrNotFound):
		writeProblem(w, r, http.StatusNotFound, "not_found", nil, nil)
	case errors.Is(err, finance.ErrNotRefundable):
		writeProblem(w, r, http.StatusConflict, "not_refundable", nil, nil)
	case errors.Is(err, finance.ErrDisputeExists):
		writeProblem(w, r, http.StatusConflict, "dispute_exists", nil, nil)
	case errors.Is(err, finance.ErrNothingToPay):
		writeProblem(w, r, http.StatusConflict, "nothing_to_pay", nil, nil)
	case errors.Is(err, finance.ErrMerchantOnHold):
		writeProblem(w, r, http.StatusConflict, "merchant_on_hold", nil, nil)
	case errors.Is(err, finance.ErrNotPending):
		writeProblem(w, r, http.StatusConflict, "not_pending", nil, nil)
	case errors.Is(err, finance.ErrNotSent):
		writeProblem(w, r, http.StatusConflict, "not_sent", nil, nil)
	case errors.Is(err, finance.ErrDisputeClosed):
		writeProblem(w, r, http.StatusConflict, "dispute_closed", nil, nil)
	case errors.Is(err, finance.ErrInvalidResolution):
		writeProblem(w, r, http.StatusUnprocessableEntity, "invalid_resolution", nil, nil)
	case errors.Is(err, finance.ErrInvalidTransition):
		writeProblem(w, r, http.StatusUnprocessableEntity, "invalid_dispute_transition", nil, nil)
	case errors.Is(err, finance.ErrSelfApproval):
		writeProblem(w, r, http.StatusForbidden, "self_approval", nil, nil)
	default:
		a.logger.ErrorContext(r.Context(), "finance request failed",
			slog.String("request_id", middleware.GetReqID(r.Context())),
			slog.String("error", err.Error()),
		)
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", nil, nil)
	}
}

// Delegation from the router.

func (s apiServer) ReverseMerchantVoucher(w http.ResponseWriter, r *http.Request, id gen.Id) {
	if s.finance != nil {
		s.finance.ReverseMerchantVoucher(w, r, id)
		return
	}
	s.ServerInterface.ReverseMerchantVoucher(w, r, id)
}

func (s apiServer) RequestVoucherRefund(w http.ResponseWriter, r *http.Request, id gen.Id) {
	if s.finance != nil {
		s.finance.RequestVoucherRefund(w, r, id)
		return
	}
	s.ServerInterface.RequestVoucherRefund(w, r, id)
}

func (s apiServer) OpenVoucherDispute(w http.ResponseWriter, r *http.Request, id gen.Id) {
	if s.finance != nil {
		s.finance.OpenVoucherDispute(w, r, id)
		return
	}
	s.ServerInterface.OpenVoucherDispute(w, r, id)
}

func (s apiServer) PreparePayout(w http.ResponseWriter, r *http.Request, id gen.Id) {
	if s.finance != nil {
		s.finance.PreparePayout(w, r, id)
		return
	}
	s.ServerInterface.PreparePayout(w, r, id)
}

func (s apiServer) ApprovePayout(w http.ResponseWriter, r *http.Request, id gen.Id) {
	if s.finance != nil {
		s.finance.ApprovePayout(w, r, id)
		return
	}
	s.ServerInterface.ApprovePayout(w, r, id)
}

func (s apiServer) MarkRefundPaid(w http.ResponseWriter, r *http.Request, id gen.Id) {
	if s.finance != nil {
		s.finance.MarkRefundPaid(w, r, id)
		return
	}
	s.ServerInterface.MarkRefundPaid(w, r, id)
}

func (s apiServer) ReconcilePayout(w http.ResponseWriter, r *http.Request, id gen.Id) {
	if s.finance != nil {
		s.finance.ReconcilePayout(w, r, id)
		return
	}
	s.ServerInterface.ReconcilePayout(w, r, id)
}

func (s apiServer) ApproveRefund(w http.ResponseWriter, r *http.Request, id gen.Id) {
	if s.finance != nil {
		s.finance.ApproveRefund(w, r, id)
		return
	}
	s.ServerInterface.ApproveRefund(w, r, id)
}

func (s apiServer) ProposeRefund(w http.ResponseWriter, r *http.Request, id gen.Id) {
	if s.finance != nil {
		s.finance.ProposeRefund(w, r, id)
		return
	}
	s.ServerInterface.ProposeRefund(w, r, id)
}

func (s apiServer) ResolveDispute(w http.ResponseWriter, r *http.Request, id gen.Id) {
	if s.finance != nil {
		s.finance.ResolveDispute(w, r, id)
		return
	}
	s.ServerInterface.ResolveDispute(w, r, id)
}

func (s apiServer) RequestDisputeEvidence(w http.ResponseWriter, r *http.Request, id gen.Id) {
	if s.finance != nil {
		s.finance.RequestDisputeEvidence(w, r, id)
		return
	}
	s.ServerInterface.RequestDisputeEvidence(w, r, id)
}

func (s apiServer) StartDisputeReview(w http.ResponseWriter, r *http.Request, id gen.Id) {
	if s.finance != nil {
		s.finance.StartDisputeReview(w, r, id)
		return
	}
	s.ServerInterface.StartDisputeReview(w, r, id)
}

func (s apiServer) CloseDispute(w http.ResponseWriter, r *http.Request, id gen.Id) {
	if s.finance != nil {
		s.finance.CloseDispute(w, r, id)
		return
	}
	s.ServerInterface.CloseDispute(w, r, id)
}

// SupportReverseVoucher undoes a redemption after the merchant window (PRD FR-05.5). Support and
// superadmin only; the reason goes to the audit trail.
func (a *FinanceAPI) SupportReverseVoucher(w http.ResponseWriter, r *http.Request, id gen.Id) {
	account, ok := a.auth.authenticate(w, r)
	if !ok {
		return
	}
	role := ""
	switch {
	case account.HasRole(identity.RoleSuperadmin):
		role = string(identity.RoleSuperadmin)
	case account.HasRole(identity.RoleSupport):
		role = string(identity.RoleSupport)
	}
	if role == "" {
		writeProblem(w, r, http.StatusForbidden, "forbidden", nil, nil)
		return
	}
	var body gen.ReverseRequest
	if _, ok := decodeBody(w, r, &body); !ok {
		return
	}
	if err := a.redemption.SupportReverse(r.Context(), account.ID, role, uuid.UUID(id), body.Reason); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s apiServer) SupportReverseVoucher(w http.ResponseWriter, r *http.Request, id gen.Id) {
	if s.finance != nil {
		s.finance.SupportReverseVoucher(w, r, id)
		return
	}
	s.ServerInterface.SupportReverseVoucher(w, r, id)
}

// GetFinanceQueue lists the open payouts, refunds and disputes for the finance desk.
func (a *FinanceAPI) GetFinanceQueue(w http.ResponseWriter, r *http.Request) {
	account, ok := a.auth.authenticate(w, r)
	if !ok {
		return
	}
	role := ""
	switch {
	case account.HasRole(identity.RoleSuperadmin):
		role = string(identity.RoleSuperadmin)
	case account.HasRole(identity.RoleFinance):
		role = string(identity.RoleFinance)
	case account.HasRole(identity.RoleSupport):
		role = string(identity.RoleSupport)
	}
	queue, err := a.finance.Queue(r.Context(), finance.Actor{UserID: account.ID, Role: role})
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, gen.FinanceQueue{
		PayoutRuns: mapQueueItems(queue.PayoutRuns, func(item finance.QueueItem) gen.PayoutRunItem {
			return gen.PayoutRunItem{Id: gen.Uuid(item.ID), MerchantName: item.Title, Amount: item.Amount, Status: item.Status, CreatedAt: item.CreatedAt}
		}),
		Refunds: mapQueueItems(queue.Refunds, func(item finance.QueueItem) gen.RefundItem {
			return gen.RefundItem{Id: gen.Uuid(item.ID), VoucherCode: item.Title, Amount: item.Amount, Status: item.Status, Reason: item.Reason, CreatedAt: item.CreatedAt}
		}),
		Disputes: mapQueueItems(queue.Disputes, func(item finance.QueueItem) gen.DisputeItem {
			return gen.DisputeItem{Id: gen.Uuid(item.ID), VoucherCode: item.Title, MerchantName: "", Amount: item.Amount, Status: item.Status, Reason: item.Reason, CreatedAt: item.CreatedAt}
		}),
	})
}

func mapQueueItems[T any](items []finance.QueueItem, mapper func(finance.QueueItem) T) []T {
	out := make([]T, 0, len(items))
	for _, item := range items {
		out = append(out, mapper(item))
	}
	return out
}

func (s apiServer) GetFinanceQueue(w http.ResponseWriter, r *http.Request) {
	if s.finance != nil {
		s.finance.GetFinanceQueue(w, r)
		return
	}
	s.ServerInterface.GetFinanceQueue(w, r)
}
