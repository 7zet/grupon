package httpapi

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/grupon/grupon/services/api/internal/domain/merchant"
	"github.com/grupon/grupon/services/api/internal/http/gen"
)

// MerchantAPI serves business registration. Other merchant-panel operations keep their generated
// 501 responses until their phase implements them.
type MerchantAPI struct {
	gen.Unimplemented

	service *merchant.Service
	auth    *AuthAPI
	logger  *slog.Logger
}

var _ gen.ServerInterface = (*MerchantAPI)(nil)

// NewMerchantAPI binds the merchant service to the HTTP contract. It needs the auth handlers to
// resolve the signed-in account.
func NewMerchantAPI(service *merchant.Service, auth *AuthAPI, logger *slog.Logger) *MerchantAPI {
	return &MerchantAPI{service: service, auth: auth, logger: logger}
}

func (a *MerchantAPI) RegisterMerchant(w http.ResponseWriter, r *http.Request) {
	account, ok := a.auth.authenticate(w, r)
	if !ok {
		return
	}
	var body gen.RegisterMerchantRequest
	if _, ok := decodeBody(w, r, &body); !ok {
		return
	}

	created, err := a.service.Register(r.Context(), account.ID, merchant.RegisterInput{
		LegalName:   body.LegalName,
		DisplayName: body.DisplayName,
		CategoryID:  uuid.UUID(body.CategoryId),
		CityID:      uuid.UUID(body.CityId),
		Phone:       body.Phone,
		Address:     body.Address,
		Lat:         float64(body.Lat),
		Lng:         float64(body.Lng),
	})
	if err != nil {
		a.failRegistration(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, gen.Merchant{
		Id:            gen.Uuid(created.ID),
		Slug:          created.Slug,
		LegalName:     created.LegalName,
		DisplayName:   created.DisplayName,
		Description:   gen.LocalizedText{},
		Phone:         created.Phone,
		Status:        gen.MerchantStatus(created.Status),
		RatingAvg:     0,
		RatingCount:   0,
		CommissionBps: 1500,
	})
}

func (a *MerchantAPI) failRegistration(w http.ResponseWriter, r *http.Request, err error) {
	var invalid *merchant.InvalidInputError
	switch {
	case errors.As(err, &invalid):
		writeProblem(w, r, http.StatusUnprocessableEntity, invalid.Code, nil,
			map[string][]string{invalid.Field: {invalid.Code}})
	case errors.Is(err, merchant.ErrCityNotFound):
		writeProblem(w, r, http.StatusUnprocessableEntity, "city_not_found", nil, nil)
	case errors.Is(err, merchant.ErrCategoryNotFound):
		writeProblem(w, r, http.StatusUnprocessableEntity, "category_not_found", nil, nil)
	default:
		a.logger.ErrorContext(r.Context(), "merchant registration failed",
			slog.String("request_id", middleware.GetReqID(r.Context())),
			slog.String("error", err.Error()),
		)
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", nil, nil)
	}
}
