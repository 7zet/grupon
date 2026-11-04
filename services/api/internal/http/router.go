// Package httpapi contains the HTTP transport and its operational endpoints.
package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/grupon/grupon/services/api/internal/http/gen"
)

const requestIDHeader = "X-Request-ID"

// ReadinessCheck reports whether the process can serve traffic.
type ReadinessCheck func(context.Context) error

// RouterOptions supplies infrastructure dependencies to the HTTP transport.
type RouterOptions struct {
	Logger         *slog.Logger
	ReadinessCheck ReadinessCheck
	// Catalog serves the public catalog endpoints when set.
	Catalog *CatalogAPI
	// Internal serves operational endpoints protected by a shared token when set.
	Internal *InternalAPI
	// Auth serves authentication and account endpoints when set.
	Auth *AuthAPI
	// Commerce serves cart, orders, payments, vouchers and saved deals when set.
	Commerce *CommerceAPI
	// Merchant serves business registration when set.
	Merchant *MerchantAPI
	// MerchantWorkspace serves the merchant overview, business profile and branches when set.
	MerchantWorkspace *MerchantWorkspaceAPI
	// MerchantMedia serves the media library when set.
	MerchantMedia *MerchantMediaAPI
	// MerchantDeals serves the merchant deal workflow when set.
	MerchantDeals *MerchantDealAPI
	// Moderation serves the review queue for platform staff when set.
	Moderation *ModerationAPI
	// MerchantReview serves the business review queue for platform staff when set.
	MerchantReview *MerchantReviewAPI
	// MerchantDocuments serves business document upload and review downloads when set.
	MerchantDocuments *MerchantDocumentAPI
	// MerchantStaff manages the business's managers and cashiers when set.
	MerchantStaff *MerchantStaffAPI
	// MerchantPanel serves payouts, customers, analytics, reviews, orders and settings when set.
	MerchantPanel *MerchantPanelAPI
	// MerchantRedemption serves voucher lookup and redemption at the counter when set.
	MerchantRedemption *MerchantRedemptionAPI
	// Reviews serves buyer reviews and review reports when set.
	Reviews *ReviewAPI
	// AdminReports serves review reports, hiding and the audit trail when set.
	AdminReports *AdminReportsAPI
	// Finance serves reversals, refunds, disputes and payouts when set.
	Finance *FinanceAPI
	// MediaFiles serves stored media files for the local storage adapter when set.
	MediaFiles http.Handler
}

// maxRequestBodyBytes caps every request body. The largest accepted payload is a small JSON object.
const maxRequestBodyBytes = 64 << 10

// NewRouter creates the application router. Business routes are mounted from their handlers.
func NewRouter(options RouterOptions) chi.Router {
	logger := options.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	readinessCheck := options.ReadinessCheck
	if readinessCheck == nil {
		readinessCheck = func(context.Context) error { return nil }
	}

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(addRequestIDHeader)
	router.Use(logRequests(logger))
	router.Use(recoverPanics(logger))
	router.Use(limitRequestBody(maxRequestBodyBytes))
	if options.Auth != nil {
		router.Use(options.Auth.withCredentials)
	}

	router.Get("/healthz", statusHandler(http.StatusOK, "ok"))
	router.Get("/readyz", readinessHandler(readinessCheck))

	if options.Catalog != nil || options.Auth != nil || options.Commerce != nil || options.Merchant != nil || options.MerchantWorkspace != nil || options.MerchantMedia != nil || options.MerchantDeals != nil || options.Moderation != nil || options.MerchantReview != nil || options.MerchantDocuments != nil || options.MerchantStaff != nil || options.MerchantPanel != nil || options.MerchantRedemption != nil || options.Reviews != nil || options.AdminReports != nil || options.Finance != nil {
		var base gen.ServerInterface = gen.Unimplemented{}
		if options.Catalog != nil {
			base = options.Catalog
		}
		gen.HandlerWithOptions(apiServer{ServerInterface: base, auth: options.Auth, commerce: options.Commerce, merchant: options.Merchant, workspace: options.MerchantWorkspace, media: options.MerchantMedia, deals: options.MerchantDeals, moderation: options.Moderation, merchantReview: options.MerchantReview, merchantDocs: options.MerchantDocuments, staff: options.MerchantStaff, panel: options.MerchantPanel, redeem: options.MerchantRedemption, reviewAPI: options.Reviews, adminReports: options.AdminReports, finance: options.Finance}, gen.ChiServerOptions{
			BaseRouter:       router,
			ErrorHandlerFunc: bindingErrorHandler,
		})
	}
	if options.Internal != nil {
		options.Internal.mount(router)
	}
	if options.MediaFiles != nil {
		router.Handle("/media/*", options.MediaFiles)
	}

	return router
}

func readinessHandler(check ReadinessCheck) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		if err := check(request.Context()); err != nil {
			writeStatus(response, http.StatusServiceUnavailable, "not_ready")
			return
		}

		writeStatus(response, http.StatusOK, "ok")
	}
}

func statusHandler(code int, status string) http.HandlerFunc {
	return func(response http.ResponseWriter, _ *http.Request) {
		writeStatus(response, code, status)
	}
}

func writeStatus(response http.ResponseWriter, code int, status string) {
	response.Header().Set("Cache-Control", "no-store")
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(code)
	_ = json.NewEncoder(response).Encode(struct {
		Status string `json:"status"`
	}{Status: status})
}

func addRequestIDHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if requestID := middleware.GetReqID(request.Context()); requestID != "" {
			response.Header().Set(requestIDHeader, requestID)
		}

		next.ServeHTTP(response, request)
	})
}

func logRequests(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			startedAt := time.Now()
			wrapped := middleware.NewWrapResponseWriter(response, request.ProtoMajor)

			next.ServeHTTP(wrapped, request)

			status := wrapped.Status()
			if status == 0 {
				status = http.StatusOK
			}
			logger.InfoContext(request.Context(), "http request completed",
				slog.String("request_id", middleware.GetReqID(request.Context())),
				slog.String("method", request.Method),
				slog.String("path", request.URL.Path),
				slog.Int("status", status),
				slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
			)
		})
	}
}

func recoverPanics(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.ErrorContext(request.Context(), "http panic recovered",
						slog.String("request_id", middleware.GetReqID(request.Context())),
					)
					writeStatus(response, http.StatusInternalServerError, "internal_error")
				}
			}()

			next.ServeHTTP(response, request)
		})
	}
}

// apiServer combines the generated operations of every handler. Operations owned by AuthAPI are
// delegated to it; everything else is served by the base handler (the catalog, or 501 responses).
type apiServer struct {
	gen.ServerInterface
	auth           *AuthAPI
	commerce       *CommerceAPI
	merchant       *MerchantAPI
	workspace      *MerchantWorkspaceAPI
	media          *MerchantMediaAPI
	deals          *MerchantDealAPI
	moderation     *ModerationAPI
	merchantReview *MerchantReviewAPI
	merchantDocs   *MerchantDocumentAPI
	staff          *MerchantStaffAPI
	panel          *MerchantPanelAPI
	redeem         *MerchantRedemptionAPI
	reviewAPI      *ReviewAPI
	adminReports   *AdminReportsAPI
	finance        *FinanceAPI
}

func (s apiServer) ListRecentlyViewed(w http.ResponseWriter, r *http.Request) {
	if s.commerce != nil {
		s.commerce.ListRecentlyViewed(w, r)
		return
	}
	s.ServerInterface.ListRecentlyViewed(w, r)
}

func (s apiServer) RecordRecentlyViewed(w http.ResponseWriter, r *http.Request) {
	if s.commerce != nil {
		s.commerce.RecordRecentlyViewed(w, r)
		return
	}
	s.ServerInterface.RecordRecentlyViewed(w, r)
}

func (s apiServer) CreateOrder(w http.ResponseWriter, r *http.Request, params gen.CreateOrderParams) {
	if s.commerce != nil {
		s.commerce.CreateOrder(w, r, params)
		return
	}
	s.ServerInterface.CreateOrder(w, r, params)
}

func (s apiServer) ListMyOrders(w http.ResponseWriter, r *http.Request, params gen.ListMyOrdersParams) {
	if s.commerce != nil {
		s.commerce.ListMyOrders(w, r, params)
		return
	}
	s.ServerInterface.ListMyOrders(w, r, params)
}

func (s apiServer) GetMyOrder(w http.ResponseWriter, r *http.Request, id gen.Id) {
	if s.commerce != nil {
		s.commerce.GetMyOrder(w, r, id)
		return
	}
	s.ServerInterface.GetMyOrder(w, r, id)
}

func (s apiServer) PayOrder(w http.ResponseWriter, r *http.Request, id gen.Id) {
	if s.commerce != nil {
		s.commerce.PayOrder(w, r, id)
		return
	}
	s.ServerInterface.PayOrder(w, r, id)
}

func (s apiServer) CompleteMockPayment(w http.ResponseWriter, r *http.Request, id gen.Id) {
	if s.commerce != nil {
		s.commerce.CompleteMockPayment(w, r, id)
		return
	}
	s.ServerInterface.CompleteMockPayment(w, r, id)
}

func (s apiServer) ReceivePaymentWebhook(w http.ResponseWriter, r *http.Request, provider gen.ReceivePaymentWebhookParamsProvider, params gen.ReceivePaymentWebhookParams) {
	if s.commerce != nil {
		s.commerce.ReceivePaymentWebhook(w, r, provider, params)
		return
	}
	s.ServerInterface.ReceivePaymentWebhook(w, r, provider, params)
}

func (s apiServer) ListMyVouchers(w http.ResponseWriter, r *http.Request, params gen.ListMyVouchersParams) {
	if s.commerce != nil {
		s.commerce.ListMyVouchers(w, r, params)
		return
	}
	s.ServerInterface.ListMyVouchers(w, r, params)
}

func (s apiServer) GetMyVoucher(w http.ResponseWriter, r *http.Request, id gen.Id) {
	if s.commerce != nil {
		s.commerce.GetMyVoucher(w, r, id)
		return
	}
	s.ServerInterface.GetMyVoucher(w, r, id)
}

func (s apiServer) RefreshSession(w http.ResponseWriter, r *http.Request) {
	if s.auth != nil {
		s.auth.RefreshSession(w, r)
		return
	}
	s.ServerInterface.RefreshSession(w, r)
}

func (s apiServer) Logout(w http.ResponseWriter, r *http.Request) {
	if s.auth != nil {
		s.auth.Logout(w, r)
		return
	}
	s.ServerInterface.Logout(w, r)
}

func (s apiServer) EnrollTwoFactor(w http.ResponseWriter, r *http.Request) {
	s.auth.EnrollTwoFactor(w, r)
}
func (s apiServer) EnableTwoFactor(w http.ResponseWriter, r *http.Request) {
	s.auth.EnableTwoFactor(w, r)
}
func (s apiServer) VerifyTwoFactor(w http.ResponseWriter, r *http.Request) {
	s.auth.VerifyTwoFactor(w, r)
}
func (s apiServer) DisableTwoFactor(w http.ResponseWriter, r *http.Request) {
	s.auth.DisableTwoFactor(w, r)
}

func (s apiServer) GetMe(w http.ResponseWriter, r *http.Request) {
	if s.auth != nil {
		s.auth.GetMe(w, r)
		return
	}
	s.ServerInterface.GetMe(w, r)
}

func (s apiServer) UpdateMe(w http.ResponseWriter, r *http.Request) {
	if s.auth != nil {
		s.auth.UpdateMe(w, r)
		return
	}
	s.ServerInterface.UpdateMe(w, r)
}

func (s apiServer) DeleteMe(w http.ResponseWriter, r *http.Request) {
	if s.auth != nil {
		s.auth.DeleteMe(w, r)
		return
	}
	s.ServerInterface.DeleteMe(w, r)
}

// RegisterMerchant is owned by MerchantAPI when it is mounted.
func (s apiServer) RegisterMerchant(w http.ResponseWriter, r *http.Request) {
	if s.merchant != nil {
		s.merchant.RegisterMerchant(w, r)
		return
	}
	s.ServerInterface.RegisterMerchant(w, r)
}

func (s apiServer) GetMerchantBusiness(w http.ResponseWriter, r *http.Request) {
	if s.workspace != nil {
		s.workspace.GetMerchantBusiness(w, r)
		return
	}
	s.ServerInterface.GetMerchantBusiness(w, r)
}

func (s apiServer) UpdateMerchantBusiness(w http.ResponseWriter, r *http.Request) {
	if s.workspace != nil {
		s.workspace.UpdateMerchantBusiness(w, r)
		return
	}
	s.ServerInterface.UpdateMerchantBusiness(w, r)
}

func (s apiServer) GetMerchantDashboard(w http.ResponseWriter, r *http.Request, params gen.GetMerchantDashboardParams) {
	if s.workspace != nil {
		s.workspace.GetMerchantDashboard(w, r, params)
		return
	}
	s.ServerInterface.GetMerchantDashboard(w, r, params)
}

func (s apiServer) ListMerchantDeals(w http.ResponseWriter, r *http.Request, params gen.ListMerchantDealsParams) {
	if s.deals != nil {
		s.deals.ListMerchantDeals(w, r, params)
		return
	}
	s.ServerInterface.ListMerchantDeals(w, r, params)
}

func (s apiServer) CreateMerchantDeal(w http.ResponseWriter, r *http.Request) {
	if s.deals != nil {
		s.deals.CreateMerchantDeal(w, r)
		return
	}
	s.ServerInterface.CreateMerchantDeal(w, r)
}

func (s apiServer) GetMerchantDeal(w http.ResponseWriter, r *http.Request, id gen.Id) {
	if s.deals != nil {
		s.deals.GetMerchantDeal(w, r, id)
		return
	}
	s.ServerInterface.GetMerchantDeal(w, r, id)
}

func (s apiServer) UpdateMerchantDeal(w http.ResponseWriter, r *http.Request, id gen.Id) {
	if s.deals != nil {
		s.deals.UpdateMerchantDeal(w, r, id)
		return
	}
	s.ServerInterface.UpdateMerchantDeal(w, r, id)
}

func (s apiServer) DeleteMerchantDeal(w http.ResponseWriter, r *http.Request, id gen.Id) {
	if s.deals != nil {
		s.deals.DeleteMerchantDeal(w, r, id)
		return
	}
	s.ServerInterface.DeleteMerchantDeal(w, r, id)
}

func (s apiServer) SubmitMerchantDeal(w http.ResponseWriter, r *http.Request, id gen.Id) {
	if s.deals != nil {
		s.deals.SubmitMerchantDeal(w, r, id)
		return
	}
	s.ServerInterface.SubmitMerchantDeal(w, r, id)
}

func (s apiServer) ListMerchantLocations(w http.ResponseWriter, r *http.Request) {
	if s.workspace != nil {
		s.workspace.ListMerchantLocations(w, r)
		return
	}
	s.ServerInterface.ListMerchantLocations(w, r)
}

func (s apiServer) CreateMerchantLocation(w http.ResponseWriter, r *http.Request) {
	if s.workspace != nil {
		s.workspace.CreateMerchantLocation(w, r)
		return
	}
	s.ServerInterface.CreateMerchantLocation(w, r)
}

func (s apiServer) UpdateMerchantLocation(w http.ResponseWriter, r *http.Request, id gen.Id) {
	if s.workspace != nil {
		s.workspace.UpdateMerchantLocation(w, r, id)
		return
	}
	s.ServerInterface.UpdateMerchantLocation(w, r, id)
}

func (s apiServer) DeleteMerchantLocation(w http.ResponseWriter, r *http.Request, id gen.Id) {
	if s.workspace != nil {
		s.workspace.DeleteMerchantLocation(w, r, id)
		return
	}
	s.ServerInterface.DeleteMerchantLocation(w, r, id)
}

func (s apiServer) CreateMerchantMediaUpload(w http.ResponseWriter, r *http.Request) {
	if s.media != nil {
		s.media.CreateMerchantMediaUpload(w, r)
		return
	}
	s.ServerInterface.CreateMerchantMediaUpload(w, r)
}

func (s apiServer) ListMerchantMedia(w http.ResponseWriter, r *http.Request) {
	if s.media != nil {
		s.media.ListMerchantMedia(w, r)
		return
	}
	s.ServerInterface.ListMerchantMedia(w, r)
}

func (s apiServer) DeleteMerchantMedia(w http.ResponseWriter, r *http.Request, id gen.Id) {
	if s.media != nil {
		s.media.DeleteMerchantMedia(w, r, id)
		return
	}
	s.ServerInterface.DeleteMerchantMedia(w, r, id)
}

func (s apiServer) UploadMerchantMediaContent(w http.ResponseWriter, r *http.Request, id gen.Id) {
	if s.media != nil {
		s.media.UploadMerchantMediaContent(w, r, id)
		return
	}
	s.ServerInterface.UploadMerchantMediaContent(w, r, id)
}

// SubmitMerchantForReview is owned by MerchantAPI when it is mounted.
func (s apiServer) SubmitMerchantForReview(w http.ResponseWriter, r *http.Request) {
	if s.merchant != nil {
		s.merchant.SubmitMerchantForReview(w, r)
		return
	}
	s.ServerInterface.SubmitMerchantForReview(w, r)
}

func (s apiServer) ListMerchantReviewQueue(w http.ResponseWriter, r *http.Request, params gen.ListMerchantReviewQueueParams) {
	if s.merchantReview != nil {
		s.merchantReview.ListMerchantReviewQueue(w, r, params)
		return
	}
	s.ServerInterface.ListMerchantReviewQueue(w, r, params)
}

func (s apiServer) StartMerchantReview(w http.ResponseWriter, r *http.Request, id gen.Id) {
	if s.merchantReview != nil {
		s.merchantReview.StartMerchantReview(w, r, id)
		return
	}
	s.ServerInterface.StartMerchantReview(w, r, id)
}

func (s apiServer) ApproveMerchant(w http.ResponseWriter, r *http.Request, id gen.Id) {
	if s.merchantReview != nil {
		s.merchantReview.ApproveMerchant(w, r, id)
		return
	}
	s.ServerInterface.ApproveMerchant(w, r, id)
}

func (s apiServer) RejectMerchant(w http.ResponseWriter, r *http.Request, id gen.Id) {
	if s.merchantReview != nil {
		s.merchantReview.RejectMerchant(w, r, id)
		return
	}
	s.ServerInterface.RejectMerchant(w, r, id)
}

func (s apiServer) ListMerchantDocuments(w http.ResponseWriter, r *http.Request) {
	if s.merchantDocs != nil {
		s.merchantDocs.ListMerchantDocuments(w, r)
		return
	}
	s.ServerInterface.ListMerchantDocuments(w, r)
}

func (s apiServer) UploadMerchantDocument(w http.ResponseWriter, r *http.Request, kind gen.UploadMerchantDocumentParamsKind) {
	if s.merchantDocs != nil {
		s.merchantDocs.UploadMerchantDocument(w, r, kind)
		return
	}
	s.ServerInterface.UploadMerchantDocument(w, r, kind)
}

func (s apiServer) DownloadMerchantDocument(w http.ResponseWriter, r *http.Request, id gen.Id, kind gen.DownloadMerchantDocumentParamsKind) {
	if s.merchantDocs != nil {
		s.merchantDocs.DownloadMerchantDocument(w, r, id, kind)
		return
	}
	s.ServerInterface.DownloadMerchantDocument(w, r, id, kind)
}

// ListMyNotifications is owned by CommerceAPI when it is mounted.
func (s apiServer) ListMyNotifications(w http.ResponseWriter, r *http.Request) {
	if s.commerce != nil {
		s.commerce.ListMyNotifications(w, r)
		return
	}
	s.ServerInterface.ListMyNotifications(w, r)
}

func (s apiServer) ListMerchantStaff(w http.ResponseWriter, r *http.Request) {
	if s.staff != nil {
		s.staff.ListMerchantStaff(w, r)
		return
	}
	s.ServerInterface.ListMerchantStaff(w, r)
}

func (s apiServer) InviteMerchantStaff(w http.ResponseWriter, r *http.Request) {
	if s.staff != nil {
		s.staff.InviteMerchantStaff(w, r)
		return
	}
	s.ServerInterface.InviteMerchantStaff(w, r)
}

func (s apiServer) UpdateMerchantStaff(w http.ResponseWriter, r *http.Request, id gen.Id) {
	if s.staff != nil {
		s.staff.UpdateMerchantStaff(w, r, id)
		return
	}
	s.ServerInterface.UpdateMerchantStaff(w, r, id)
}

func (s apiServer) RemoveMerchantStaff(w http.ResponseWriter, r *http.Request, id gen.Id) {
	if s.staff != nil {
		s.staff.RemoveMerchantStaff(w, r, id)
		return
	}
	s.ServerInterface.RemoveMerchantStaff(w, r, id)
}

func (s apiServer) ListMerchantPayouts(w http.ResponseWriter, r *http.Request, params gen.ListMerchantPayoutsParams) {
	if s.panel != nil {
		s.panel.ListMerchantPayouts(w, r, params)
		return
	}
	s.ServerInterface.ListMerchantPayouts(w, r, params)
}

func (s apiServer) ListMerchantCustomers(w http.ResponseWriter, r *http.Request, params gen.ListMerchantCustomersParams) {
	if s.panel != nil {
		s.panel.ListMerchantCustomers(w, r, params)
		return
	}
	s.ServerInterface.ListMerchantCustomers(w, r, params)
}

func (s apiServer) GetMerchantAnalytics(w http.ResponseWriter, r *http.Request, params gen.GetMerchantAnalyticsParams) {
	if s.panel != nil {
		s.panel.GetMerchantAnalytics(w, r, params)
		return
	}
	s.ServerInterface.GetMerchantAnalytics(w, r, params)
}

func (s apiServer) ListMerchantReviews(w http.ResponseWriter, r *http.Request, params gen.ListMerchantReviewsParams) {
	if s.panel != nil {
		s.panel.ListMerchantReviews(w, r, params)
		return
	}
	s.ServerInterface.ListMerchantReviews(w, r, params)
}

func (s apiServer) ReplyToMerchantReview(w http.ResponseWriter, r *http.Request, id gen.Id) {
	if s.panel != nil {
		s.panel.ReplyToMerchantReview(w, r, id)
		return
	}
	s.ServerInterface.ReplyToMerchantReview(w, r, id)
}

func (s apiServer) ListMerchantOrders(w http.ResponseWriter, r *http.Request, params gen.ListMerchantOrdersParams) {
	if s.panel != nil {
		s.panel.ListMerchantOrders(w, r, params)
		return
	}
	s.ServerInterface.ListMerchantOrders(w, r, params)
}

func (s apiServer) GetMerchantSettings(w http.ResponseWriter, r *http.Request) {
	if s.panel != nil {
		s.panel.GetMerchantSettings(w, r)
		return
	}
	s.ServerInterface.GetMerchantSettings(w, r)
}

func (s apiServer) UpdateMerchantSettings(w http.ResponseWriter, r *http.Request) {
	if s.panel != nil {
		s.panel.UpdateMerchantSettings(w, r)
		return
	}
	s.ServerInterface.UpdateMerchantSettings(w, r)
}
