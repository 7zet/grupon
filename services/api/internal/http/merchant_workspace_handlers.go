package httpapi

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/grupon/grupon/services/api/internal/domain/catalog"
	"github.com/grupon/grupon/services/api/internal/domain/merchant"
	"github.com/grupon/grupon/services/api/internal/http/gen"
)

// MerchantWorkspaceAPI serves the merchant overview, business profile and branches. Every operation
// resolves the signed-in merchant from the database; the request never names a merchant id.
type MerchantWorkspaceAPI struct {
	gen.Unimplemented

	workspace *merchant.WorkspaceService
	auth      *AuthAPI
	logger    *slog.Logger
}

var _ gen.ServerInterface = (*MerchantWorkspaceAPI)(nil)

// NewMerchantWorkspaceAPI binds the workspace service to the HTTP contract.
func NewMerchantWorkspaceAPI(workspace *merchant.WorkspaceService, auth *AuthAPI, logger *slog.Logger) *MerchantWorkspaceAPI {
	return &MerchantWorkspaceAPI{workspace: workspace, auth: auth, logger: logger}
}

// scope authenticates the caller and resolves their merchant. Cashiers are refused when mayEdit is set.
func (a *MerchantWorkspaceAPI) scope(w http.ResponseWriter, r *http.Request, mayEdit bool) (uuid.UUID, bool) {
	account, ok := a.auth.authenticate(w, r)
	if !ok {
		return uuid.Nil, false
	}
	merchantID, _, err := a.workspace.Authorize(r.Context(), account.ID, mayEdit)
	if err != nil {
		a.fail(w, r, err)
		return uuid.Nil, false
	}
	return merchantID, true
}

func (a *MerchantWorkspaceAPI) GetMerchantBusiness(w http.ResponseWriter, r *http.Request) {
	merchantID, ok := a.scope(w, r, false)
	if !ok {
		return
	}
	business, err := a.workspace.Business(r.Context(), merchantID)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapBusiness(business))
}

func (a *MerchantWorkspaceAPI) UpdateMerchantBusiness(w http.ResponseWriter, r *http.Request) {
	merchantID, ok := a.scope(w, r, true)
	if !ok {
		return
	}
	var body gen.MerchantInput
	if _, ok := decodeBody(w, r, &body); !ok {
		return
	}
	patch := merchant.BusinessPatch{
		DisplayName: body.DisplayName,
		LogoURL:     body.LogoUrl,
		CoverURL:    body.CoverUrl,
		Phone:       body.Phone,
		Website:     body.Website,
		Instagram:   body.Instagram,
		Telegram:    body.Telegram,
	}
	if body.Description != nil {
		text := catalog.Text{Uz: body.Description.Uz, Ru: body.Description.Ru, En: body.Description.En}
		patch.Description = &text
	}
	business, err := a.workspace.UpdateBusiness(r.Context(), merchantID, patch)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapBusiness(business))
}

func (a *MerchantWorkspaceAPI) GetMerchantDashboard(w http.ResponseWriter, r *http.Request, _ gen.GetMerchantDashboardParams) {
	merchantID, ok := a.scope(w, r, false)
	if !ok {
		return
	}
	dashboard, err := a.workspace.Dashboard(r.Context(), merchantID)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	chart := make([]gen.ChartPoint, 0, len(dashboard.SalesChart))
	for _, point := range dashboard.SalesChart {
		chart = append(chart, gen.ChartPoint{Date: openapi_types.Date{Time: point.Day}, Value: point.Value})
	}
	pending := make([]gen.DealSummary, 0, len(dashboard.PendingDeals))
	for _, deal := range dashboard.PendingDeals {
		pending = append(pending, mapDealSummary(deal))
	}
	writeJSON(w, http.StatusOK, gen.MerchantDashboard{
		TotalSales:       mapMetric(dashboard.TotalSales),
		VouchersSold:     mapMetric(dashboard.VouchersSold),
		VouchersRedeemed: mapMetric(dashboard.VouchersRedeemed),
		ActiveDeals:      mapMetric(dashboard.ActiveDeals),
		SalesChart:       chart,
		LatestOrders:     []gen.Order{},
		PendingDeals:     pending,
	})
}

func (a *MerchantWorkspaceAPI) ListMerchantLocations(w http.ResponseWriter, r *http.Request) {
	merchantID, ok := a.scope(w, r, false)
	if !ok {
		return
	}
	locations, err := a.workspace.Locations(r.Context(), merchantID)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, gen.LocationList{Items: mapWorkspaceLocations(locations)})
}

func (a *MerchantWorkspaceAPI) CreateMerchantLocation(w http.ResponseWriter, r *http.Request) {
	merchantID, ok := a.scope(w, r, true)
	if !ok {
		return
	}
	var body gen.LocationInput
	if _, ok := decodeBody(w, r, &body); !ok {
		return
	}
	location, err := a.workspace.CreateLocation(r.Context(), merchantID, locationInput(body))
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, mapWorkspaceLocation(location))
}

func (a *MerchantWorkspaceAPI) UpdateMerchantLocation(w http.ResponseWriter, r *http.Request, id gen.Id) {
	merchantID, ok := a.scope(w, r, true)
	if !ok {
		return
	}
	var body gen.LocationInput
	if _, ok := decodeBody(w, r, &body); !ok {
		return
	}
	if err := a.workspace.UpdateLocation(r.Context(), merchantID, uuid.UUID(id), locationInput(body)); err != nil {
		a.fail(w, r, err)
		return
	}
	locations, err := a.workspace.Locations(r.Context(), merchantID)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	for _, location := range locations {
		if location.ID == uuid.UUID(id) {
			writeJSON(w, http.StatusOK, mapWorkspaceLocation(location))
			return
		}
	}
	writeProblem(w, r, http.StatusNotFound, "location_not_found", nil, nil)
}

func (a *MerchantWorkspaceAPI) DeleteMerchantLocation(w http.ResponseWriter, r *http.Request, id gen.Id) {
	merchantID, ok := a.scope(w, r, true)
	if !ok {
		return
	}
	if err := a.workspace.DeleteLocation(r.Context(), merchantID, uuid.UUID(id)); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *MerchantWorkspaceAPI) fail(w http.ResponseWriter, r *http.Request, err error) {
	var invalid *merchant.InvalidInputError
	switch {
	case errors.As(err, &invalid):
		writeProblem(w, r, http.StatusUnprocessableEntity, invalid.Code, nil,
			map[string][]string{invalid.Field: {invalid.Code}})
	case errors.Is(err, merchant.ErrNoMerchant), errors.Is(err, merchant.ErrForbidden):
		writeProblem(w, r, http.StatusForbidden, "forbidden", nil, nil)
	case errors.Is(err, merchant.ErrNotFound):
		writeProblem(w, r, http.StatusNotFound, "not_found", nil, nil)
	case errors.Is(err, merchant.ErrLocationInUse):
		writeProblem(w, r, http.StatusConflict, "location_in_use", nil, nil)
	case errors.Is(err, merchant.ErrCityNotFound):
		writeProblem(w, r, http.StatusUnprocessableEntity, "city_not_found", nil, nil)
	default:
		a.logger.ErrorContext(r.Context(), "merchant workspace request failed",
			slog.String("request_id", middleware.GetReqID(r.Context())),
			slog.String("error", err.Error()),
		)
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", nil, nil)
	}
}

func locationInput(body gen.LocationInput) merchant.LocationInput {
	hours := map[string]any{}
	for key, value := range body.OpeningHours {
		hours[key] = value
	}
	return merchant.LocationInput{
		CityID:       uuid.UUID(body.CityId),
		Address:      body.Address,
		Lat:          float64(body.Lat),
		Lng:          float64(body.Lng),
		OpeningHours: hours,
		Phone:        body.Phone,
	}
}

func mapBusiness(business merchant.Business) gen.Merchant {
	return gen.Merchant{
		Id:            gen.Uuid(business.ID),
		Slug:          business.Slug,
		LegalName:     business.LegalName,
		DisplayName:   business.DisplayName,
		Description:   mapText(business.Description),
		LogoUrl:       business.LogoURL,
		CoverUrl:      business.CoverURL,
		Phone:         business.Phone,
		Website:       business.Website,
		Instagram:     business.Instagram,
		Telegram:      business.Telegram,
		Status:        gen.MerchantStatus(business.Status),
		ReviewStatus:  reviewStatusPtr(business.ReviewStatus),
		RatingAvg:     float32(business.RatingAvg),
		RatingCount:   business.RatingCount,
		CommissionBps: 1500,
	}
}

func mapMetric(metric merchant.Metric) gen.Metric {
	return gen.Metric{Value: metric.Value, ChangePercent: float32(metric.ChangePercent)}
}

func mapWorkspaceLocation(location merchant.Location) gen.MerchantLocation {
	return gen.MerchantLocation{
		Id:           gen.Uuid(location.ID),
		City:         mapCity(location.City),
		Address:      location.Address,
		Lat:          float32(location.Lat),
		Lng:          float32(location.Lng),
		OpeningHours: location.OpeningHours,
		Phone:        location.Phone,
	}
}

func mapWorkspaceLocations(locations []merchant.Location) []gen.MerchantLocation {
	result := make([]gen.MerchantLocation, 0, len(locations))
	for _, location := range locations {
		result = append(result, mapWorkspaceLocation(location))
	}
	return result
}
