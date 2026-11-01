package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/grupon/grupon/services/api/internal/domain/catalog"
	"github.com/grupon/grupon/services/api/internal/http/gen"
)

const defaultLocale = "uz"

// CatalogAPI serves the public catalog endpoints. Every other generated operation keeps the
// generated 501 response until its phase implements it.
type CatalogAPI struct {
	gen.Unimplemented

	catalog *catalog.Service
	logger  *slog.Logger
}

var _ gen.ServerInterface = (*CatalogAPI)(nil)

// NewCatalogAPI binds the catalog service to the HTTP contract.
func NewCatalogAPI(service *catalog.Service, logger *slog.Logger) *CatalogAPI {
	return &CatalogAPI{catalog: service, logger: logger}
}

func (a *CatalogAPI) GetPublicConfig(w http.ResponseWriter, r *http.Request) {
	config, err := a.catalog.PublicConfig(r.Context())
	if err != nil {
		a.fail(w, r, err, "")
		return
	}
	writeJSON(w, http.StatusOK, gen.PublicConfig{
		Locales:    []gen.Locale{gen.Locale("uz"), gen.Locale("ru"), gen.Locale("en")},
		Currencies: []gen.Currency{gen.Currency("UZS")},
		Cities:     mapCities(config.Cities),
		Categories: mapCategories(config.Categories),
	})
}

func (a *CatalogAPI) ListCities(w http.ResponseWriter, r *http.Request) {
	cities, err := a.catalog.Cities(r.Context())
	if err != nil {
		a.fail(w, r, err, "")
		return
	}
	writeJSON(w, http.StatusOK, gen.CityList{Items: mapCities(cities)})
}

func (a *CatalogAPI) ListCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := a.catalog.Categories(r.Context())
	if err != nil {
		a.fail(w, r, err, "")
		return
	}
	writeJSON(w, http.StatusOK, gen.CategoryList{Items: mapCategories(categories)})
}

func (a *CatalogAPI) GetHomepage(w http.ResponseWriter, r *http.Request, params gen.GetHomepageParams) {
	if _, err := resolveLocale(params.Locale); err != nil {
		a.fail(w, r, err, "")
		return
	}
	sections, err := a.catalog.Homepage(r.Context())
	if err != nil {
		a.fail(w, r, err, "")
		return
	}
	result := make([]gen.HomepageSection, 0, len(sections))
	for _, section := range sections {
		result = append(result, mapHomepageSection(section))
	}
	writeJSON(w, http.StatusOK, gen.Homepage{Sections: result})
}

func (a *CatalogAPI) ListDeals(w http.ResponseWriter, r *http.Request, params gen.ListDealsParams) {
	if _, err := resolveLocale(params.Locale); err != nil {
		a.fail(w, r, err, "")
		return
	}
	sortValue := ""
	if params.Sort != nil {
		sortValue = string(*params.Sort)
	}
	sort, err := catalog.ParseSort(sortValue)
	if err != nil {
		a.fail(w, r, err, "")
		return
	}
	page, err := catalog.NewPage(params.Page, params.PageSize)
	if err != nil {
		a.fail(w, r, err, "")
		return
	}
	query, err := catalog.SearchQuery(stringValue(params.Q))
	if err != nil {
		a.fail(w, r, err, "")
		return
	}
	if params.MinPrice != nil && params.MaxPrice != nil && *params.MinPrice > *params.MaxPrice {
		a.fail(w, r, &catalog.InvalidInputError{Field: "max_price", Code: "invalid_price_range"}, "")
		return
	}
	filter := catalog.DealFilter{
		City:        stringValue(params.City),
		Category:    stringValue(params.Category),
		Subcategory: stringValue(params.Subcategory),
		Query:       query,
		MinPrice:    params.MinPrice,
		MaxPrice:    params.MaxPrice,
		MinDiscount: params.MinDiscount,
		EndingSoon:  boolValue(params.EndingSoon),
	}
	if params.MinRating != nil {
		rating := float64(*params.MinRating)
		filter.MinRating = &rating
	}
	result, err := a.catalog.ListDeals(r.Context(), filter, sort, page)
	if err != nil {
		a.fail(w, r, err, "")
		return
	}
	writeJSON(w, http.StatusOK, mapDealPage(result))
}

func (a *CatalogAPI) GetDeal(w http.ResponseWriter, r *http.Request, slug gen.Slug, params gen.GetDealParams) {
	if _, err := resolveLocale(params.Locale); err != nil {
		a.fail(w, r, err, "")
		return
	}
	deal, err := a.catalog.Deal(r.Context(), slug)
	if err != nil {
		a.fail(w, r, err, "deal_not_found")
		return
	}
	writeJSON(w, http.StatusOK, mapDeal(deal))
}

func (a *CatalogAPI) ListRelatedDeals(w http.ResponseWriter, r *http.Request, slug gen.Slug, params gen.ListRelatedDealsParams) {
	if _, err := resolveLocale(params.Locale); err != nil {
		a.fail(w, r, err, "")
		return
	}
	related, err := a.catalog.RelatedDeals(r.Context(), slug, stringValue(params.City))
	if err != nil {
		a.fail(w, r, err, "deal_not_found")
		return
	}
	writeJSON(w, http.StatusOK, gen.DealList{Items: mapDealSummaries(related.Items)})
}

func (a *CatalogAPI) ListDealReviews(w http.ResponseWriter, r *http.Request, slug gen.Slug, params gen.ListDealReviewsParams) {
	if _, err := resolveLocale(params.Locale); err != nil {
		a.fail(w, r, err, "")
		return
	}
	page, err := catalog.NewPage(params.Page, params.PageSize)
	if err != nil {
		a.fail(w, r, err, "")
		return
	}
	reviews, err := a.catalog.DealReviews(r.Context(), slug, page)
	if err != nil {
		a.fail(w, r, err, "deal_not_found")
		return
	}
	writeJSON(w, http.StatusOK, gen.ReviewList{Items: mapReviews(reviews.Items)})
}

func (a *CatalogAPI) GetBusiness(w http.ResponseWriter, r *http.Request, slug gen.Slug, params gen.GetBusinessParams) {
	if _, err := resolveLocale(params.Locale); err != nil {
		a.fail(w, r, err, "")
		return
	}
	profile, err := a.catalog.Merchant(r.Context(), slug)
	if err != nil {
		a.fail(w, r, err, "merchant_not_found")
		return
	}
	writeJSON(w, http.StatusOK, mapMerchantProfile(profile))
}

func (a *CatalogAPI) GetSearchSuggestions(w http.ResponseWriter, r *http.Request, params gen.GetSearchSuggestionsParams) {
	locale, err := resolveLocale(params.Locale)
	if err != nil {
		a.fail(w, r, err, "")
		return
	}
	suggestions, err := a.catalog.Suggestions(r.Context(), params.Q, locale)
	if err != nil {
		a.fail(w, r, err, "")
		return
	}
	items := make([]gen.Suggestion, 0, len(suggestions))
	for _, suggestion := range suggestions {
		items = append(items, gen.Suggestion{
			Type:  gen.SuggestionType(suggestion.Type),
			Label: suggestion.Label,
			Href:  suggestion.Href,
		})
	}
	writeJSON(w, http.StatusOK, gen.SuggestionList{Items: items})
}

// fail translates a domain error into an RFC 9457 problem. notFoundCode names the missing resource
// for the frontend; an empty value falls back to the generic not_found code.
func (a *CatalogAPI) fail(w http.ResponseWriter, r *http.Request, err error, notFoundCode string) {
	var invalidErr *catalog.InvalidInputError
	switch {
	case errors.As(err, &invalidErr):
		writeProblem(w, r, http.StatusBadRequest, invalidErr.Code, nil, map[string][]string{invalidErr.Field: {invalidErr.Code}})
	case errors.Is(err, catalog.ErrNotFound):
		code := notFoundCode
		if code == "" {
			code = "not_found"
		}
		writeProblem(w, r, http.StatusNotFound, code, nil, nil)
	default:
		a.logger.ErrorContext(r.Context(), "catalog request failed",
			slog.String("request_id", middleware.GetReqID(r.Context())),
			slog.String("error", err.Error()),
		)
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", nil, nil)
	}
}

func resolveLocale(locale *gen.Locale) (string, error) {
	if locale == nil {
		return defaultLocale, nil
	}
	switch string(*locale) {
	case "uz", "ru", "en":
		return string(*locale), nil
	default:
		return "", &catalog.InvalidInputError{Field: "locale", Code: "invalid_locale"}
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func boolValue(value *bool) bool {
	return value != nil && *value
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeProblem(w http.ResponseWriter, r *http.Request, status int, code string, detail *string, errs map[string][]string) {
	problem := gen.Problem{
		Type:     "about:blank",
		Title:    http.StatusText(status),
		Status:   status,
		Code:     code,
		Instance: stringPtr(r.URL.Path),
		Detail:   detail,
	}
	if len(errs) > 0 {
		problem.Errors = &errs
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(problem)
}

func stringPtr(value string) *string {
	return &value
}

func mapText(text catalog.Text) gen.LocalizedText {
	return gen.LocalizedText{Uz: text.Uz, Ru: text.Ru, En: text.En}
}

func mapTexts(texts []catalog.Text) []gen.LocalizedText {
	result := make([]gen.LocalizedText, 0, len(texts))
	for _, text := range texts {
		result = append(result, mapText(text))
	}
	return result
}

func mapCities(cities []catalog.City) []gen.City {
	result := make([]gen.City, 0, len(cities))
	for _, city := range cities {
		result = append(result, mapCity(city))
	}
	return result
}

func mapCity(city catalog.City) gen.City {
	return gen.City{
		Id:   gen.Uuid(city.ID),
		Slug: city.Slug,
		Name: mapText(city.Name),
		Lat:  float32(city.Lat),
		Lng:  float32(city.Lng),
	}
}

func mapCategories(categories []catalog.Category) []gen.Category {
	result := make([]gen.Category, 0, len(categories))
	for _, category := range categories {
		result = append(result, mapCategory(category))
	}
	return result
}

func mapCategory(category catalog.Category) gen.Category {
	subcategories := make([]gen.Subcategory, 0, len(category.Subcategories))
	for _, sub := range category.Subcategories {
		subcategories = append(subcategories, gen.Subcategory{Id: gen.Uuid(sub.ID), Slug: sub.Slug, Name: mapText(sub.Name)})
	}
	return gen.Category{
		Id:            gen.Uuid(category.ID),
		Slug:          category.Slug,
		Name:          mapText(category.Name),
		Icon:          category.Icon,
		Subcategories: subcategories,
	}
}

func mapMerchant(merchant catalog.Merchant) gen.Merchant {
	result := gen.Merchant{
		Id:            gen.Uuid(merchant.ID),
		Slug:          merchant.Slug,
		LegalName:     merchant.LegalName,
		DisplayName:   merchant.DisplayName,
		Description:   mapText(merchant.Description),
		LogoUrl:       merchant.LogoURL,
		CoverUrl:      merchant.CoverURL,
		Phone:         merchant.Phone,
		Website:       merchant.Website,
		Instagram:     merchant.Instagram,
		Telegram:      merchant.Telegram,
		Status:        gen.MerchantStatus("active"),
		RatingAvg:     float32(merchant.RatingAvg),
		RatingCount:   merchant.RatingCount,
		CommissionBps: merchant.CommissionBPS,
	}
	return result
}

func mapLocations(locations []catalog.Location) []gen.MerchantLocation {
	result := make([]gen.MerchantLocation, 0, len(locations))
	for _, location := range locations {
		result = append(result, gen.MerchantLocation{
			Id:           gen.Uuid(location.ID),
			City:         mapCity(location.City),
			Address:      location.Address,
			Lat:          float32(location.Lat),
			Lng:          float32(location.Lng),
			OpeningHours: location.OpeningHours,
			Phone:        location.Phone,
		})
	}
	return result
}

func mapOptions(options []catalog.Option) []gen.DealOption {
	result := make([]gen.DealOption, 0, len(options))
	for _, option := range options {
		result = append(result, gen.DealOption{
			Id:            gen.Uuid(option.ID),
			Title:         mapText(option.Title),
			OriginalPrice: mapMoney(option.OriginalPrice),
			DealPrice:     mapMoney(option.DealPrice),
			StockTotal:    option.StockTotal,
			StockSold:     option.StockSold,
			StockReserved: option.StockReserved,
			IsActive:      option.IsActive,
		})
	}
	return result
}

func mapMoney(money catalog.Money) gen.Money {
	return gen.Money{Amount: money.Amount, Currency: gen.MoneyCurrency(catalog.Currency)}
}

func mapMedia(items []catalog.Media) []gen.DealMedia {
	result := make([]gen.DealMedia, 0, len(items))
	for _, item := range items {
		result = append(result, gen.DealMedia{
			Id:        gen.Uuid(item.ID),
			Url:       item.URL,
			Alt:       mapText(item.Alt),
			SortOrder: item.SortOrder,
			Width:     item.Width,
			Height:    item.Height,
			Blurhash:  item.Blurhash,
		})
	}
	return result
}

func mapDealSummary(deal catalog.DealSummary) gen.DealSummary {
	return gen.DealSummary{
		Id:          gen.Uuid(deal.ID),
		Slug:        deal.Slug,
		Merchant:    mapMerchant(deal.Merchant),
		Category:    mapCategory(deal.Category),
		Title:       mapText(deal.Title),
		Status:      gen.DealStatus(deal.Status),
		Options:     mapOptions(deal.Options),
		Media:       mapMedia(deal.Media),
		SoldCount:   deal.SoldCount,
		RatingAvg:   float32(deal.RatingAvg),
		RatingCount: deal.RatingCount,
		StartsAt:    deal.StartsAt,
		EndsAt:      deal.EndsAt,
	}
}

func mapDealSummaries(deals []catalog.DealSummary) []gen.DealSummary {
	result := make([]gen.DealSummary, 0, len(deals))
	for _, deal := range deals {
		result = append(result, mapDealSummary(deal))
	}
	return result
}

func mapDealPage(page catalog.DealPage) gen.DealPage {
	return gen.DealPage{
		Items:    mapDealSummaries(page.Items),
		Page:     page.Page,
		PageSize: page.PageSize,
		Total:    page.Total,
	}
}

func mapDeal(deal catalog.Deal) gen.Deal {
	faq := make([]gen.FaqItem, 0, len(deal.FAQ))
	for _, item := range deal.FAQ {
		faq = append(faq, gen.FaqItem{Q: mapText(item.Q), A: mapText(item.A)})
	}
	var instructions *gen.LocalizedText
	if deal.RedemptionInstructions != nil {
		mapped := mapText(*deal.RedemptionInstructions)
		instructions = &mapped
	}
	locations := mapLocations(deal.Locations)
	return gen.Deal{
		Id:                     gen.Uuid(deal.ID),
		Slug:                   deal.Slug,
		Merchant:               mapMerchant(deal.Merchant),
		Category:               mapCategory(deal.Category),
		Subcategory:            gen.Subcategory{Id: gen.Uuid(deal.Subcategory.ID), Slug: deal.Subcategory.Slug, Name: mapText(deal.Subcategory.Name)},
		Title:                  mapText(deal.Title),
		ShortDescription:       mapText(deal.ShortDescription),
		Description:            mapText(deal.Description),
		Highlights:             mapTexts(deal.Highlights),
		WhatsIncluded:          mapTexts(deal.WhatsIncluded),
		Terms:                  mapText(deal.Terms),
		Faq:                    faq,
		RedemptionMethod:       gen.DealRedemptionMethod(deal.RedemptionMethod),
		RedemptionInstructions: instructions,
		BookingRequired:        deal.BookingRequired,
		VoucherValidDays:       deal.VoucherValidDays,
		VoucherValidUntil:      deal.VoucherValidUntil,
		MaxPerUser:             deal.MaxPerUser,
		Status:                 gen.DealStatus(deal.Status),
		Options:                mapOptions(deal.Options),
		Media:                  mapMedia(deal.Media),
		SoldCount:              deal.SoldCount,
		RatingAvg:              float32(deal.RatingAvg),
		RatingCount:            deal.RatingCount,
		StartsAt:               deal.StartsAt,
		EndsAt:                 deal.EndsAt,
		Locations:              locations,
	}
}

func mapReviews(reviews []catalog.Review) []gen.Review {
	result := make([]gen.Review, 0, len(reviews))
	for _, review := range reviews {
		item := gen.Review{
			Id:         gen.Uuid(review.ID),
			DealId:     gen.Uuid(review.DealID),
			MerchantId: gen.Uuid(review.MerchantID),
			UserId:     gen.Uuid(review.UserID),
			VoucherId:  gen.Uuid(review.VoucherID),
			Rating:     review.Rating,
			Text:       review.Text,
			Locale:     gen.Locale(review.Locale),
			Status:     gen.ReviewStatus(review.Status),
			CreatedAt:  review.CreatedAt,
		}
		if review.Reply != nil {
			item.Reply = &gen.ReviewReply{
				Id:        gen.Uuid(review.Reply.ID),
				Text:      review.Reply.Text,
				CreatedAt: review.Reply.CreatedAt,
			}
		}
		result = append(result, item)
	}
	return result
}

func mapMerchantProfile(profile catalog.MerchantProfile) gen.MerchantProfile {
	locations := mapLocations(profile.Locations)
	return gen.MerchantProfile{
		Id:            gen.Uuid(profile.ID),
		Slug:          profile.Slug,
		LegalName:     profile.LegalName,
		DisplayName:   profile.DisplayName,
		Description:   mapText(profile.Description),
		LogoUrl:       profile.LogoURL,
		CoverUrl:      profile.CoverURL,
		Phone:         profile.Phone,
		Website:       profile.Website,
		Instagram:     profile.Instagram,
		Telegram:      profile.Telegram,
		Status:        gen.MerchantProfileStatus("active"),
		RatingAvg:     float32(profile.RatingAvg),
		RatingCount:   profile.RatingCount,
		CommissionBps: profile.CommissionBPS,
		Locations:     &locations,
		Deals:         mapDealSummaries(profile.Deals),
		Reviews:       mapReviews(profile.Reviews),
	}
}

func mapHomepageSection(section catalog.HomepageSection) gen.HomepageSection {
	result := gen.HomepageSection{
		Id:        gen.Uuid(section.ID),
		Key:       section.Key,
		Title:     mapText(section.Title),
		Type:      gen.HomepageSectionType(section.Type),
		SortOrder: section.SortOrder,
	}
	if section.Deals != nil {
		deals := mapDealSummaries(section.Deals)
		result.Deals = &deals
	}
	if section.Categories != nil {
		categories := mapCategories(section.Categories)
		result.Categories = &categories
	}
	return result
}
