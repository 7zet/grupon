package httpapi

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/grupon/grupon/services/api/internal/domain/merchant"
	"github.com/grupon/grupon/services/api/internal/http/gen"
)

// MerchantPanelAPI serves the merchant panel pages that read and answer business data: payouts,
// customers, analytics, reviews, orders and settings. Rules live in the merchant domain.
type MerchantPanelAPI struct {
	service   *merchant.PanelService
	workspace *merchant.WorkspaceService
	auth      *AuthAPI
	logger    *slog.Logger
}

// NewMerchantPanelAPI binds the panel rules to the HTTP contract.
func NewMerchantPanelAPI(service *merchant.PanelService, workspace *merchant.WorkspaceService, auth *AuthAPI, logger *slog.Logger) *MerchantPanelAPI {
	return &MerchantPanelAPI{service: service, workspace: workspace, auth: auth, logger: logger}
}

func (a *MerchantPanelAPI) scope(w http.ResponseWriter, r *http.Request, mayEdit bool) (merchant.Actor, uuid.UUID, bool) {
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

func (a *MerchantPanelAPI) ListMerchantPayouts(w http.ResponseWriter, r *http.Request, _ gen.ListMerchantPayoutsParams) {
	_, merchantID, ok := a.scope(w, r, false)
	if !ok {
		return
	}
	rows, err := a.service.Payouts(r.Context(), merchantID)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	items := make([]gen.Payout, 0, len(rows))
	for _, row := range rows {
		items = append(items, gen.Payout{
			Id:          gen.Uuid(row.ID),
			PeriodStart: openapi_types.Date{Time: row.PeriodStart},
			PeriodEnd:   openapi_types.Date{Time: row.PeriodEnd},
			Gross:       money(row.Gross),
			Commission:  money(row.Commission),
			Net:         money(row.Net),
			Status:      gen.PayoutStatus(row.Status),
			PaidAt:      row.PaidAt,
		})
	}
	writeJSON(w, http.StatusOK, gen.PayoutPage{Items: items})
}

func (a *MerchantPanelAPI) ListMerchantCustomers(w http.ResponseWriter, r *http.Request, _ gen.ListMerchantCustomersParams) {
	_, merchantID, ok := a.scope(w, r, false)
	if !ok {
		return
	}
	rows, err := a.service.Customers(r.Context(), merchantID)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	items := make([]gen.MerchantCustomer, 0, len(rows))
	for _, row := range rows {
		items = append(items, gen.MerchantCustomer{
			UserId:        gen.Uuid(row.UserID),
			Name:          row.Name,
			OrdersCount:   row.OrdersCount,
			VouchersCount: row.VouchersCount,
			TotalSpent:    money(row.TotalSpent),
			LastOrderAt:   row.LastOrderAt,
		})
	}
	writeJSON(w, http.StatusOK, gen.CustomerPage{Items: items})
}

func (a *MerchantPanelAPI) GetMerchantAnalytics(w http.ResponseWriter, r *http.Request, params gen.GetMerchantAnalyticsParams) {
	_, merchantID, ok := a.scope(w, r, false)
	if !ok {
		return
	}
	// The end date is inclusive in the contract; the store takes the exclusive end of that day.
	from := params.From.Time
	to := params.To.Time.AddDate(0, 0, 1)
	out, err := a.service.Analytics(r.Context(), merchantID, from, to)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	sales := make([]gen.ChartPoint, 0, len(out.Sales))
	for _, point := range out.Sales {
		sales = append(sales, gen.ChartPoint{Date: openapi_types.Date{Time: point.Date}, Value: point.Amount})
	}
	writeJSON(w, http.StatusOK, gen.MerchantAnalytics{
		GrossSales:       money(out.GrossSales),
		VouchersSold:     int(out.VouchersSold),
		VouchersRedeemed: int(out.VouchersRedeem),
		RedemptionRate:   float32(out.RedemptionRate),
		Sales:            sales,
		TopDeals:         []gen.DealSummary{},
	})
}

func (a *MerchantPanelAPI) ListMerchantReviews(w http.ResponseWriter, r *http.Request, _ gen.ListMerchantReviewsParams) {
	_, merchantID, ok := a.scope(w, r, false)
	if !ok {
		return
	}
	rows, err := a.service.Reviews(r.Context(), merchantID)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	items := make([]gen.Review, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapPanelReview(row, merchantID))
	}
	writeJSON(w, http.StatusOK, gen.ReviewList{Items: items})
}

func (a *MerchantPanelAPI) ReplyToMerchantReview(w http.ResponseWriter, r *http.Request, id gen.Id) {
	actor, merchantID, ok := a.scope(w, r, true)
	if !ok {
		return
	}
	var body gen.ReplyRequest
	if _, ok := decodeBody(w, r, &body); !ok {
		return
	}
	reply, err := a.service.Reply(r.Context(), actor, merchantID, uuid.UUID(id), body.Text)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, gen.ReviewReply{Id: gen.Uuid(reply.ID), Text: reply.Text, CreatedAt: reply.CreatedAt})
}

func (a *MerchantPanelAPI) ListMerchantOrders(w http.ResponseWriter, r *http.Request, _ gen.ListMerchantOrdersParams) {
	_, merchantID, ok := a.scope(w, r, false)
	if !ok {
		return
	}
	rows, err := a.service.Orders(r.Context(), merchantID)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	items := make([]gen.Order, 0, len(rows))
	for _, row := range rows {
		items = append(items, gen.Order{
			Id:        gen.Uuid(row.ID),
			Number:    row.Number,
			Status:    gen.OrderStatus(row.Status),
			Subtotal:  money(row.Subtotal),
			Total:     money(row.Total),
			Items:     []gen.OrderItem{},
			CreatedAt: row.CreatedAt,
		})
	}
	writeJSON(w, http.StatusOK, gen.OrderPage{Items: items})
}

func (a *MerchantPanelAPI) GetMerchantSettings(w http.ResponseWriter, r *http.Request) {
	_, merchantID, ok := a.scope(w, r, false)
	if !ok {
		return
	}
	settings, err := a.service.Settings(r.Context(), merchantID)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapSettings(settings))
}

func (a *MerchantPanelAPI) UpdateMerchantSettings(w http.ResponseWriter, r *http.Request) {
	actor, merchantID, ok := a.scope(w, r, true)
	if !ok {
		return
	}
	var body gen.MerchantSettings
	if _, ok := decodeBody(w, r, &body); !ok {
		return
	}
	settings := merchant.Settings{OrderEmails: body.OrderEmails, ReviewEmails: body.ReviewEmails, PayoutEmails: body.PayoutEmails}
	if body.DefaultLocationId != nil {
		id := uuid.UUID(*body.DefaultLocationId)
		settings.DefaultLocationID = &id
	}
	saved, err := a.service.UpdateSettings(r.Context(), actor, merchantID, settings)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapSettings(saved))
}

func (a *MerchantPanelAPI) fail(w http.ResponseWriter, r *http.Request, err error) {
	var invalid *merchant.InvalidInputError
	switch {
	case errors.As(err, &invalid):
		writeProblem(w, r, http.StatusUnprocessableEntity, invalid.Code, nil,
			map[string][]string{invalid.Field: {invalid.Code}})
	case errors.Is(err, merchant.ErrReviewNotFound), errors.Is(err, merchant.ErrNotFound):
		writeProblem(w, r, http.StatusNotFound, "not_found", nil, nil)
	case errors.Is(err, merchant.ErrReplyLocked):
		writeProblem(w, r, http.StatusConflict, "reply_locked", nil, nil)
	case errors.Is(err, merchant.ErrForbidden):
		writeProblem(w, r, http.StatusForbidden, "forbidden", nil, nil)
	default:
		a.logger.ErrorContext(r.Context(), "merchant panel failed",
			slog.String("request_id", middleware.GetReqID(r.Context())),
			slog.String("error", err.Error()),
		)
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", nil, nil)
	}
}

func money(amount int64) gen.Money {
	return gen.Money{Amount: amount, Currency: gen.MoneyCurrencyUZS}
}

func mapSettings(s merchant.Settings) gen.MerchantSettings {
	out := gen.MerchantSettings{OrderEmails: s.OrderEmails, ReviewEmails: s.ReviewEmails, PayoutEmails: s.PayoutEmails}
	if s.DefaultLocationID != nil {
		id := gen.Uuid(*s.DefaultLocationID)
		out.DefaultLocationId = &id
	}
	return out
}

func mapPanelReview(row merchant.ReviewRow, merchantID uuid.UUID) gen.Review {
	out := gen.Review{
		Id:         gen.Uuid(row.ID),
		DealId:     gen.Uuid(row.DealID),
		MerchantId: gen.Uuid(merchantID),
		UserId:     gen.Uuid(row.UserID),
		VoucherId:  gen.Uuid(row.VoucherID),
		Rating:     row.Rating,
		Text:       row.Text,
		Locale:     gen.Locale(row.Locale),
		Status:     gen.ReviewStatus(row.Status),
		CreatedAt:  row.CreatedAt,
	}
	if row.Reply != nil {
		out.Reply = &gen.ReviewReply{Id: gen.Uuid(row.Reply.ID), Text: row.Reply.Text, CreatedAt: row.Reply.CreatedAt}
	}
	return out
}
