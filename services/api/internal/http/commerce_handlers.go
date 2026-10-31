package httpapi

import (
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/grupon/grupon/services/api/internal/domain/catalog"
	"github.com/grupon/grupon/services/api/internal/domain/commerce"
	"github.com/grupon/grupon/services/api/internal/http/gen"
	"github.com/grupon/grupon/services/api/internal/payments"
)

const (
	defaultPageSize = 24
	maxWebhookBytes = 16 << 10
)

// CommerceAPI serves cart, saved deals, orders, payments, vouchers and the payment webhook.
// Handlers translate HTTP to the commerce service and back; they hold no business rules.
type CommerceAPI struct {
	gen.Unimplemented

	service *commerce.Service
	auth    *AuthAPI
	logger  *slog.Logger
}

// NewCommerceAPI binds the commerce service to the HTTP contract. Authentication reuses AuthAPI.
func NewCommerceAPI(service *commerce.Service, auth *AuthAPI, logger *slog.Logger) *CommerceAPI {
	return &CommerceAPI{service: service, auth: auth, logger: logger}
}

// GetCart returns the stored cart.
func (a *CommerceAPI) GetCart(w http.ResponseWriter, r *http.Request) {
	account, ok := a.auth.authenticate(w, r)
	if !ok {
		return
	}
	cart, err := a.service.GetCart(r.Context(), account.ID)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapCart(cart))
}

// ReplaceCart stores the cart, or merges it into the stored cart when merge is set.
func (a *CommerceAPI) ReplaceCart(w http.ResponseWriter, r *http.Request) {
	account, ok := a.auth.authenticate(w, r)
	if !ok {
		return
	}
	var body gen.CartInput
	if _, ok := decodeBody(w, r, &body); !ok {
		return
	}
	lines := make([]commerce.CartLine, 0, len(body.Items))
	for _, item := range body.Items {
		lines = append(lines, commerce.CartLine{OptionID: uuid.UUID(item.DealOptionId), Quantity: item.Quantity, IsGift: item.IsGift})
	}
	merge := body.Merge != nil && *body.Merge
	cart, err := a.service.ReplaceCart(r.Context(), account.ID, lines, merge)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapCart(cart))
}

// ListSavedDeals returns the saved deals.
func (a *CommerceAPI) ListSavedDeals(w http.ResponseWriter, r *http.Request) {
	account, ok := a.auth.authenticate(w, r)
	if !ok {
		return
	}
	deals, err := a.service.SavedDeals(r.Context(), account.ID)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, gen.DealList{Items: mapDealSummaries(deals)})
}

// SaveDeal saves a deal.
func (a *CommerceAPI) SaveDeal(w http.ResponseWriter, r *http.Request) {
	account, ok := a.auth.authenticate(w, r)
	if !ok {
		return
	}
	var body gen.DealIdRequest
	if _, ok := decodeBody(w, r, &body); !ok {
		return
	}
	if err := a.service.SaveDeal(r.Context(), account.ID, uuid.UUID(body.DealId)); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// UnsaveDeal removes a saved deal by deal id.
func (a *CommerceAPI) UnsaveDeal(w http.ResponseWriter, r *http.Request, id gen.Id) {
	account, ok := a.auth.authenticate(w, r)
	if !ok {
		return
	}
	if err := a.service.UnsaveDeal(r.Context(), account.ID, uuid.UUID(id)); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListRecentlyViewed returns the recently viewed deals.
func (a *CommerceAPI) ListRecentlyViewed(w http.ResponseWriter, r *http.Request) {
	account, ok := a.auth.authenticate(w, r)
	if !ok {
		return
	}
	deals, err := a.service.RecentlyViewed(r.Context(), account.ID)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, gen.DealList{Items: mapDealSummaries(deals)})
}

// RecordRecentlyViewed records a view of a deal.
func (a *CommerceAPI) RecordRecentlyViewed(w http.ResponseWriter, r *http.Request) {
	account, ok := a.auth.authenticate(w, r)
	if !ok {
		return
	}
	var body gen.DealIdRequest
	if _, ok := decodeBody(w, r, &body); !ok {
		return
	}
	if err := a.service.RecordView(r.Context(), account.ID, uuid.UUID(body.DealId)); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// CreateOrder creates a pending order and its stock reservations. A replay of the same key and body
// answers 200 with the original order; a new order answers 201.
func (a *CommerceAPI) CreateOrder(w http.ResponseWriter, r *http.Request, params gen.CreateOrderParams) {
	account, ok := a.auth.authenticate(w, r)
	if !ok {
		return
	}
	var body gen.CreateOrderRequest
	if _, ok := decodeBody(w, r, &body); !ok {
		return
	}
	input := commerce.OrderInput{IdempotencyKey: params.IdempotencyKey}
	for _, item := range body.Items {
		input.Items = append(input.Items, commerce.OrderItemInput{
			OptionID:           uuid.UUID(item.DealOptionId),
			Quantity:           item.Quantity,
			IsGift:             item.IsGift,
			GiftRecipientName:  item.GiftRecipientName,
			GiftRecipientEmail: emailString(item.GiftRecipientEmail),
			GiftMessage:        item.GiftMessage,
		})
	}
	if body.ShippingAddress != nil {
		input.ShippingAddress = *body.ShippingAddress
	}
	order, replayed, err := a.service.CreateOrder(r.Context(), account.ID, input)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	status := http.StatusCreated
	if replayed {
		status = http.StatusOK
	}
	writeJSON(w, status, mapOrder(order))
}

// ListMyOrders returns the account's orders, newest first.
func (a *CommerceAPI) ListMyOrders(w http.ResponseWriter, r *http.Request, params gen.ListMyOrdersParams) {
	account, ok := a.auth.authenticate(w, r)
	if !ok {
		return
	}
	page, pageSize := pagination(params.Page, params.PageSize)
	orders, total, err := a.service.ListOrders(r.Context(), account.ID, page, pageSize)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	items := make([]gen.Order, 0, len(orders))
	for _, order := range orders {
		items = append(items, mapOrder(order))
	}
	writeJSON(w, http.StatusOK, gen.OrderPage{Items: items, Page: page, PageSize: pageSize, Total: total})
}

// GetMyOrder returns one order of the account.
func (a *CommerceAPI) GetMyOrder(w http.ResponseWriter, r *http.Request, id gen.Id) {
	account, ok := a.auth.authenticate(w, r)
	if !ok {
		return
	}
	order, err := a.service.GetOrder(r.Context(), account.ID, uuid.UUID(id))
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapOrder(order))
}

// PayOrder starts a payment attempt for an order.
func (a *CommerceAPI) PayOrder(w http.ResponseWriter, r *http.Request, id gen.Id) {
	account, ok := a.auth.authenticate(w, r)
	if !ok {
		return
	}
	var body gen.PayOrderRequest
	if _, ok := decodeBody(w, r, &body); !ok {
		return
	}
	session, err := a.service.PayOrder(r.Context(), account.ID, uuid.UUID(id), string(body.Provider))
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, gen.PaymentSession{
		PaymentId:   gen.Uuid(session.PaymentID),
		Provider:    gen.PaymentSessionProvider(session.Provider),
		Status:      gen.PaymentSessionStatus(session.Status),
		RedirectUrl: session.RedirectURL,
	})
}

// CompleteMockPayment simulates the buyer's choice on the mock checkout page. The outcome travels
// through the same signed webhook path a real provider uses. It answers 202 once applied.
func (a *CommerceAPI) CompleteMockPayment(w http.ResponseWriter, r *http.Request, id gen.Id) {
	var body gen.MockOutcomeRequest
	if _, ok := decodeBody(w, r, &body); !ok {
		return
	}
	if err := a.service.CompleteMockPayment(r.Context(), uuid.UUID(id), payments.Outcome(body.Outcome)); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

// ReceivePaymentWebhook verifies and applies a provider webhook. The signature is checked before
// the body is interpreted; the handler is idempotent per provider payment.
func (a *CommerceAPI) ReceivePaymentWebhook(w http.ResponseWriter, r *http.Request, provider gen.ReceivePaymentWebhookParamsProvider, params gen.ReceivePaymentWebhookParams) {
	raw, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBytes+1))
	if err != nil || len(raw) > maxWebhookBytes {
		writeProblem(w, r, http.StatusRequestEntityTooLarge, "payload_too_large", nil, nil)
		return
	}
	if err := a.service.HandleWebhook(r.Context(), string(provider), raw, params.XWebhookSignature); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListMyVouchers returns the account's vouchers, optionally filtered by status.
func (a *CommerceAPI) ListMyVouchers(w http.ResponseWriter, r *http.Request, params gen.ListMyVouchersParams) {
	account, ok := a.auth.authenticate(w, r)
	if !ok {
		return
	}
	status := ""
	if params.Status != nil {
		status = string(*params.Status)
	}
	page, pageSize := pagination(params.Page, nil)
	views, total, err := a.service.ListVouchers(r.Context(), account.ID, status, page, pageSize)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	items := make([]gen.Voucher, 0, len(views))
	for _, view := range views {
		items = append(items, mapVoucher(view))
	}
	writeJSON(w, http.StatusOK, gen.VoucherPage{Items: items, Page: page, PageSize: pageSize, Total: total})
}

// GetMyVoucher returns one voucher of the account, including its QR token.
func (a *CommerceAPI) GetMyVoucher(w http.ResponseWriter, r *http.Request, id gen.Id) {
	account, ok := a.auth.authenticate(w, r)
	if !ok {
		return
	}
	view, err := a.service.GetVoucher(r.Context(), account.ID, uuid.UUID(id))
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapVoucher(view))
}

// fail maps commerce errors to stable problem codes. Unknown errors are logged, not returned.
func (a *CommerceAPI) fail(w http.ResponseWriter, r *http.Request, err error) {
	var validation *commerce.ValidationError
	switch {
	case errors.As(err, &validation):
		writeProblem(w, r, http.StatusUnprocessableEntity, "validation_failed", nil,
			map[string][]string{validation.Field: {validation.Code}})
	case errors.Is(err, commerce.ErrOptionUnavailable):
		writeProblem(w, r, http.StatusUnprocessableEntity, "option_unavailable", nil, nil)
	case errors.Is(err, commerce.ErrGiftNotAllowed):
		writeProblem(w, r, http.StatusUnprocessableEntity, "gift_not_allowed", nil, nil)
	case errors.Is(err, commerce.ErrMaxPerUserExceeded):
		writeProblem(w, r, http.StatusUnprocessableEntity, "max_per_user_exceeded", nil, nil)
	case errors.Is(err, commerce.ErrShippingAddressRequired):
		writeProblem(w, r, http.StatusUnprocessableEntity, "shipping_address_required", nil, nil)
	case errors.Is(err, commerce.ErrIdempotencyConflict):
		writeProblem(w, r, http.StatusUnprocessableEntity, "idempotency_conflict", nil, nil)
	case errors.Is(err, commerce.ErrInsufficientStock):
		writeProblem(w, r, http.StatusConflict, "insufficient_stock", nil, nil)
	case errors.Is(err, commerce.ErrReservationExpired):
		writeProblem(w, r, http.StatusConflict, "reservation_expired", nil, nil)
	case errors.Is(err, commerce.ErrOrderNotPayable):
		writeProblem(w, r, http.StatusConflict, "order_not_payable", nil, nil)
	case errors.Is(err, commerce.ErrPaymentNotPending):
		writeProblem(w, r, http.StatusConflict, "payment_not_pending", nil, nil)
	case errors.Is(err, commerce.ErrInvalidSignature):
		writeProblem(w, r, http.StatusUnauthorized, "invalid_signature", nil, nil)
	case errors.Is(err, commerce.ErrMalformedWebhook):
		writeProblem(w, r, http.StatusBadRequest, "invalid_webhook", nil, nil)
	case errors.Is(err, commerce.ErrProviderUnavailable):
		writeProblem(w, r, http.StatusNotFound, "payment_provider_unavailable", nil, nil)
	case errors.Is(err, commerce.ErrOrderNotFound):
		writeProblem(w, r, http.StatusNotFound, "order_not_found", nil, nil)
	case errors.Is(err, commerce.ErrPaymentNotFound):
		writeProblem(w, r, http.StatusNotFound, "payment_not_found", nil, nil)
	case errors.Is(err, commerce.ErrVoucherNotFound):
		writeProblem(w, r, http.StatusNotFound, "voucher_not_found", nil, nil)
	case errors.Is(err, commerce.ErrDealNotFound):
		writeProblem(w, r, http.StatusNotFound, "deal_not_found", nil, nil)
	case errors.Is(err, commerce.ErrNotFound), errors.Is(err, commerce.ErrUserNotFound):
		writeProblem(w, r, http.StatusNotFound, "not_found", nil, nil)
	default:
		a.logger.ErrorContext(r.Context(), "commerce request failed",
			slog.String("request_id", middleware.GetReqID(r.Context())),
			slog.String("error", err.Error()),
		)
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", nil, nil)
	}
}

func pagination(page *gen.Page, pageSize *gen.PageSize) (int, int) {
	number := 1
	if page != nil {
		number = int(*page)
	}
	size := defaultPageSize
	if pageSize != nil {
		size = int(*pageSize)
	}
	return number, size
}

func emailString(value *openapi_types.Email) *string {
	if value == nil {
		return nil
	}
	text := string(*value)
	return &text
}

func emailValue(value *string) *openapi_types.Email {
	if value == nil {
		return nil
	}
	email := openapi_types.Email(*value)
	return &email
}

func mapCart(cart commerce.Cart) gen.Cart {
	items := make([]gen.CartItem, 0, len(cart.Items))
	for _, item := range cart.Items {
		items = append(items, gen.CartItem{
			Deal:         mapDealSummary(item.Deal),
			DealOptionId: gen.Uuid(item.Line.OptionID),
			IsGift:       item.Line.IsGift,
			LineTotal:    gen.Money{Amount: item.LineTotal, Currency: gen.MoneyCurrencyUZS},
			Option:       mapOptionRecord(item.Option),
			Quantity:     item.Line.Quantity,
		})
	}
	return gen.Cart{Items: items, Subtotal: gen.Money{Amount: cart.Subtotal, Currency: gen.MoneyCurrencyUZS}}
}

func mapOptionRecord(option commerce.OptionRecord) gen.DealOption {
	return gen.DealOption{
		Id:            gen.Uuid(option.ID),
		Title:         mapText(option.Title),
		OriginalPrice: gen.Money{Amount: option.OriginalPrice, Currency: gen.MoneyCurrencyUZS},
		DealPrice:     gen.Money{Amount: option.DealPrice, Currency: gen.MoneyCurrencyUZS},
		StockTotal:    option.StockTotal,
		StockSold:     option.StockSold,
		StockReserved: option.StockReserved,
		IsActive:      option.IsActive,
	}
}

func mapOrder(order commerce.Order) gen.Order {
	items := make([]gen.OrderItem, 0, len(order.Items))
	for _, item := range order.Items {
		isGift := item.IsGift
		items = append(items, gen.OrderItem{
			Id:                 gen.Uuid(item.ID),
			DealId:             gen.Uuid(item.DealID),
			DealOptionId:       gen.Uuid(item.OptionID),
			TitleSnapshot:      mapText(item.Title),
			UnitPrice:          gen.Money{Amount: item.UnitPrice, Currency: gen.MoneyCurrencyUZS},
			OriginalPrice:      gen.Money{Amount: item.OriginalPrice, Currency: gen.MoneyCurrencyUZS},
			Quantity:           item.Quantity,
			IsGift:             &isGift,
			GiftRecipientName:  item.GiftRecipientName,
			GiftRecipientEmail: emailValue(item.GiftRecipientEmail),
			GiftMessage:        item.GiftMessage,
			FulfillmentStatus:  gen.OrderItemFulfillmentStatus(item.FulfillmentStatus),
		})
	}
	var shipping *map[string]interface{}
	if order.ShippingAddress != nil {
		shipping = &order.ShippingAddress
	}
	return gen.Order{
		Id:                   gen.Uuid(order.ID),
		Number:               order.Number,
		Status:               gen.OrderStatus(order.Status),
		Subtotal:             gen.Money{Amount: order.Subtotal, Currency: gen.MoneyCurrencyUZS},
		Total:                gen.Money{Amount: order.Total, Currency: gen.MoneyCurrencyUZS},
		ShippingAddress:      shipping,
		Items:                items,
		ReservationExpiresAt: order.ReservationExpiresAt,
		PaidAt:               order.PaidAt,
		CreatedAt:            order.CreatedAt,
	}
}

func mapVoucher(view commerce.VoucherView) gen.Voucher {
	merchant := mapMerchant(merchantOf(view.Deal))
	return gen.Voucher{
		Id:         gen.Uuid(view.Voucher.ID),
		Code:       view.Voucher.Code,
		QrToken:    view.Voucher.QRToken,
		Status:     gen.VoucherStatus(view.Voucher.Status),
		ValidFrom:  view.Voucher.ValidFrom,
		ValidUntil: view.Voucher.ValidUntil,
		RedeemedAt: view.Voucher.RedeemedAt,
		Deal:       mapDealSummary(view.Deal),
		Merchant:   merchant,
		Option:     mapOptionRecord(view.Option),
	}
}

func merchantOf(deal catalog.DealSummary) catalog.Merchant {
	return deal.Merchant
}
