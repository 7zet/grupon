package httpapi

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/grupon/grupon/services/api/internal/domain/catalog"
	"github.com/grupon/grupon/services/api/internal/domain/merchant"
	"github.com/grupon/grupon/services/api/internal/http/gen"
)

// MerchantDealAPI serves the merchant deal workflow: draft, edit, submit and delete.
// Cashiers can read deals but cannot change them; the domain enforces that as well.
type MerchantDealAPI struct {
	gen.Unimplemented

	deals     *merchant.DealService
	workspace *merchant.WorkspaceService
	auth      *AuthAPI
	logger    *slog.Logger
}

var _ gen.ServerInterface = (*MerchantDealAPI)(nil)

// NewMerchantDealAPI binds the deal workflow to the HTTP contract.
func NewMerchantDealAPI(deals *merchant.DealService, workspace *merchant.WorkspaceService, auth *AuthAPI, logger *slog.Logger) *MerchantDealAPI {
	return &MerchantDealAPI{deals: deals, workspace: workspace, auth: auth, logger: logger}
}

// scope authenticates the caller and resolves their merchant and role.
func (a *MerchantDealAPI) scope(w http.ResponseWriter, r *http.Request, mayEdit bool) (merchant.Actor, uuid.UUID, bool) {
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

func (a *MerchantDealAPI) ListMerchantDeals(w http.ResponseWriter, r *http.Request, params gen.ListMerchantDealsParams) {
	_, merchantID, ok := a.scope(w, r, false)
	if !ok {
		return
	}
	page, err := catalog.NewPage(params.Page, nil)
	if err != nil {
		writeProblem(w, r, http.StatusUnprocessableEntity, "invalid_page", nil, map[string][]string{"page": {"invalid_page"}})
		return
	}
	status := ""
	if params.Status != nil {
		status = string(*params.Status)
	}
	list, err := a.deals.List(r.Context(), merchantID, status, page)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	items := make([]gen.DealSummary, 0, len(list.Items))
	for _, item := range list.Items {
		items = append(items, mapDealSummary(item))
	}
	writeJSON(w, http.StatusOK, gen.DealPage{Items: items, Page: list.Page.Number, PageSize: list.Page.Size, Total: list.Total})
}

func (a *MerchantDealAPI) CreateMerchantDeal(w http.ResponseWriter, r *http.Request) {
	actor, merchantID, ok := a.scope(w, r, true)
	if !ok {
		return
	}
	var body gen.DealInput
	if _, ok := decodeBody(w, r, &body); !ok {
		return
	}
	deal, err := a.deals.Create(r.Context(), merchantID, actor, dealInput(body))
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, mapDeal(deal))
}

func (a *MerchantDealAPI) GetMerchantDeal(w http.ResponseWriter, r *http.Request, id gen.Id) {
	_, merchantID, ok := a.scope(w, r, false)
	if !ok {
		return
	}
	deal, err := a.deals.Get(r.Context(), merchantID, uuid.UUID(id))
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapDeal(deal))
}

func (a *MerchantDealAPI) UpdateMerchantDeal(w http.ResponseWriter, r *http.Request, id gen.Id) {
	actor, merchantID, ok := a.scope(w, r, true)
	if !ok {
		return
	}
	var body gen.DealInput
	if _, ok := decodeBody(w, r, &body); !ok {
		return
	}
	deal, err := a.deals.Update(r.Context(), merchantID, uuid.UUID(id), actor, dealInput(body))
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapDeal(deal))
}

func (a *MerchantDealAPI) DeleteMerchantDeal(w http.ResponseWriter, r *http.Request, id gen.Id) {
	actor, merchantID, ok := a.scope(w, r, true)
	if !ok {
		return
	}
	if err := a.deals.Delete(r.Context(), merchantID, uuid.UUID(id), actor); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *MerchantDealAPI) SubmitMerchantDeal(w http.ResponseWriter, r *http.Request, id gen.Id) {
	actor, merchantID, ok := a.scope(w, r, true)
	if !ok {
		return
	}
	deal, err := a.deals.Submit(r.Context(), merchantID, uuid.UUID(id), actor)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapDeal(deal))
}

func (a *MerchantDealAPI) fail(w http.ResponseWriter, r *http.Request, err error) {
	var invalid *merchant.InvalidInputError
	switch {
	case errors.As(err, &invalid):
		writeProblem(w, r, http.StatusUnprocessableEntity, invalid.Code, nil,
			map[string][]string{invalid.Field: {invalid.Code}})
	case errors.Is(err, merchant.ErrNoMerchant), errors.Is(err, merchant.ErrForbidden):
		writeProblem(w, r, http.StatusForbidden, "forbidden", nil, nil)
	case errors.Is(err, merchant.ErrDealNotFound), errors.Is(err, merchant.ErrNotFound):
		writeProblem(w, r, http.StatusNotFound, "not_found", nil, nil)
	case errors.Is(err, merchant.ErrDealStateInvalid):
		writeProblem(w, r, http.StatusConflict, "invalid_deal_state", nil, nil)
	default:
		a.logger.ErrorContext(r.Context(), "merchant deal request failed",
			slog.String("request_id", middleware.GetReqID(r.Context())),
			slog.String("error", err.Error()),
		)
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", nil, nil)
	}
}

// dealInput maps the contract body to the domain draft. Optional fields keep their zero values.
func dealInput(body gen.DealInput) merchant.DealInput {
	in := merchant.DealInput{
		Slug:                   body.Slug,
		CategoryID:             uuid.UUID(body.CategoryId),
		SubcategoryID:          uuid.UUID(body.SubcategoryId),
		Title:                  textOf(body.Title),
		ShortDescription:       textOf(body.ShortDescription),
		Description:            textOf(body.Description),
		Highlights:             textsOf(body.Highlights),
		WhatsIncluded:          textsOf(body.WhatsIncluded),
		Terms:                  textOf(body.Terms),
		RedemptionMethod:       string(body.RedemptionMethod),
		RedemptionInstructions: textOf(body.RedemptionInstructions),
		BookingRequired:        body.BookingRequired,
		StartsAt:               body.StartsAt,
		EndsAt:                 body.EndsAt,
		VoucherValidDays:       body.VoucherValidDays,
		VoucherValidUntil:      body.VoucherValidUntil,
		MaxPerUser:             body.MaxPerUser,
		FAQ:                    make([]catalog.FAQItem, 0, len(body.Faq)),
		LocationIDs:            make([]uuid.UUID, 0, len(body.LocationIds)),
		Options:                make([]merchant.DealOptionData, 0, len(body.Options)),
		Media:                  make([]merchant.DealMediaData, 0, len(body.Media)),
	}
	for _, item := range body.Faq {
		in.FAQ = append(in.FAQ, catalog.FAQItem{Q: textOf(item.Q), A: textOf(item.A)})
	}
	for _, id := range body.LocationIds {
		in.LocationIDs = append(in.LocationIDs, uuid.UUID(id))
	}
	for _, option := range body.Options {
		data := merchant.DealOptionData{
			Title:         textOf(option.Title),
			OriginalPrice: option.OriginalPrice,
			DealPrice:     option.DealPrice,
			StockTotal:    option.StockTotal,
			SortOrder:     option.SortOrder,
			IsActive:      option.IsActive,
		}
		if option.RedemptionsAllowed != nil {
			data.RedemptionsAllowed = *option.RedemptionsAllowed
		}
		if option.Id != nil {
			id := uuid.UUID(*option.Id)
			data.ID = &id
		}
		in.Options = append(in.Options, data)
	}
	for _, item := range body.Media {
		in.Media = append(in.Media, merchant.DealMediaData{
			URL:       item.Url,
			Alt:       textOf(item.Alt),
			SortOrder: item.SortOrder,
			Width:     item.Width,
			Height:    item.Height,
			Blurhash:  item.Blurhash,
		})
	}
	return in
}

func textOf(text gen.LocalizedText) catalog.Text {
	return catalog.Text{Uz: text.Uz, Ru: text.Ru, En: text.En}
}

func textsOf(items []gen.LocalizedText) []catalog.Text {
	result := make([]catalog.Text, 0, len(items))
	for _, item := range items {
		result = append(result, textOf(item))
	}
	return result
}
