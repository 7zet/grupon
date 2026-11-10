package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/dealo/dealo/services/api/internal/domain/catalog"
	sqlc "github.com/dealo/dealo/services/api/internal/store/sqlc"
)

// CatalogRepository implements the public catalog read model on top of sqlc queries.
type CatalogRepository struct {
	queries *sqlc.Queries
}

// NewCatalogRepository wraps a connection pool or transaction for catalog reads.
func NewCatalogRepository(db sqlc.DBTX) *CatalogRepository {
	return &CatalogRepository{queries: sqlc.New(db)}
}

var _ catalog.Repository = (*CatalogRepository)(nil)

func (r *CatalogRepository) ListCities(ctx context.Context) ([]catalog.City, error) {
	rows, err := r.queries.ListActiveCities(ctx)
	if err != nil {
		return nil, fmt.Errorf("list cities: %w", err)
	}
	cities := make([]catalog.City, 0, len(rows))
	for _, row := range rows {
		name, err := catalog.ParseText(row.Name)
		if err != nil {
			return nil, err
		}
		cities = append(cities, catalog.City{ID: toUUID(row.ID), Slug: row.Slug, Name: name, Lat: row.Lat, Lng: row.Lng})
	}
	return cities, nil
}

func (r *CatalogRepository) ListCategories(ctx context.Context) ([]catalog.Category, error) {
	rows, err := r.queries.ListActiveCategories(ctx)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	subRows, err := r.queries.ListSubcategories(ctx)
	if err != nil {
		return nil, fmt.Errorf("list subcategories: %w", err)
	}
	byCategory := make(map[uuid.UUID][]catalog.Subcategory)
	for _, row := range subRows {
		name, err := catalog.ParseText(row.Name)
		if err != nil {
			return nil, err
		}
		categoryID := toUUID(row.CategoryID)
		byCategory[categoryID] = append(byCategory[categoryID], catalog.Subcategory{ID: toUUID(row.ID), Slug: row.Slug, Name: name})
	}
	categories := make([]catalog.Category, 0, len(rows))
	for _, row := range rows {
		name, err := catalog.ParseText(row.Name)
		if err != nil {
			return nil, err
		}
		id := toUUID(row.ID)
		subcategories := byCategory[id]
		if subcategories == nil {
			subcategories = []catalog.Subcategory{}
		}
		categories = append(categories, catalog.Category{ID: id, Slug: row.Slug, Name: name, Icon: row.Icon,
			Searchable: row.Searchable, DealCreationEnabled: row.DealCreationEnabled, PublishEnabled: row.PublishEnabled,
			Purchasable: row.Purchasable, Redeemable: row.Redeemable, Subcategories: subcategories})
	}
	return categories, nil
}

func (r *CatalogRepository) SearchDealIDs(ctx context.Context, filter catalog.DealFilter, sort catalog.Sort, page catalog.Page) ([]uuid.UUID, error) {
	rows, err := r.queries.SearchPublishedDealIDs(ctx, sqlc.SearchPublishedDealIDsParams{
		Sort:        string(sort),
		PageOffset:  int32(page.Offset()),
		PageLimit:   int32(page.Size),
		Merchant:    optText(filter.Merchant),
		Category:    optText(filter.Category),
		Subcategory: optText(filter.Subcategory),
		City:        optText(filter.City),
		Query:       optText(filter.Query),
		MinPrice:    optInt64(filter.MinPrice),
		MaxPrice:    optInt64(filter.MaxPrice),
		MinRating:   optFloat64(filter.MinRating),
		MinDiscount: optInt32(filter.MinDiscount),
		EndingSoon:  filter.EndingSoon,
	})
	if err != nil {
		return nil, fmt.Errorf("search deal ids: %w", err)
	}
	return toUUIDs(rows), nil
}

func (r *CatalogRepository) CountDeals(ctx context.Context, filter catalog.DealFilter) (int64, error) {
	total, err := r.queries.CountPublishedDeals(ctx, sqlc.CountPublishedDealsParams{
		Merchant:    optText(filter.Merchant),
		Category:    optText(filter.Category),
		Subcategory: optText(filter.Subcategory),
		City:        optText(filter.City),
		Query:       optText(filter.Query),
		MinPrice:    optInt64(filter.MinPrice),
		MaxPrice:    optInt64(filter.MaxPrice),
		MinRating:   optFloat64(filter.MinRating),
		MinDiscount: optInt32(filter.MinDiscount),
		EndingSoon:  filter.EndingSoon,
	})
	if err != nil {
		return 0, fmt.Errorf("count deals: %w", err)
	}
	return total, nil
}

// DealSummaries loads card data for the deals in one batch: summary rows, options and media.
func (r *CatalogRepository) DealSummaries(ctx context.Context, ids []uuid.UUID) ([]catalog.DealSummary, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	dealIDs := toPgUUIDs(ids)
	rows, err := r.queries.ListDealSummaryRows(ctx, dealIDs)
	if err != nil {
		return nil, fmt.Errorf("list deal summary rows: %w", err)
	}
	optionRows, err := r.queries.ListActiveDealOptionsForDeals(ctx, dealIDs)
	if err != nil {
		return nil, fmt.Errorf("list deal options: %w", err)
	}
	mediaRows, err := r.queries.ListDealMediaForDeals(ctx, dealIDs)
	if err != nil {
		return nil, fmt.Errorf("list deal media: %w", err)
	}

	options := make(map[uuid.UUID][]catalog.Option)
	for _, row := range optionRows {
		option, err := optionFromRow(row)
		if err != nil {
			return nil, err
		}
		dealID := toUUID(row.DealID)
		options[dealID] = append(options[dealID], option)
	}
	media := make(map[uuid.UUID][]catalog.Media)
	for _, row := range mediaRows {
		item, err := mediaFromRow(row)
		if err != nil {
			return nil, err
		}
		dealID := toUUID(row.DealID)
		media[dealID] = append(media[dealID], item)
	}

	summaries := make([]catalog.DealSummary, 0, len(rows))
	for _, row := range rows {
		summary, err := summaryFromRow(row)
		if err != nil {
			return nil, err
		}
		summary.Options = nonNil(options[summary.ID])
		summary.Media = nonNil(media[summary.ID])
		summaries = append(summaries, summary)
	}
	return summaries, nil
}

func (r *CatalogRepository) DealBySlug(ctx context.Context, slug string) (catalog.Deal, error) {
	detailRow, err := r.queries.GetDealDetailRowBySlug(ctx, slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return catalog.Deal{}, catalog.ErrNotFound
	}
	if err != nil {
		return catalog.Deal{}, fmt.Errorf("get deal: %w", err)
	}
	dealID := toUUID(detailRow.ID)
	summaries, err := r.DealSummaries(ctx, []uuid.UUID{dealID})
	if err != nil {
		return catalog.Deal{}, err
	}
	if len(summaries) != 1 {
		return catalog.Deal{}, catalog.ErrNotFound
	}
	locationRows, err := r.queries.ListDealLocationRows(ctx, []pgtype.UUID{toPgUUID(dealID)})
	if err != nil {
		return catalog.Deal{}, fmt.Errorf("list deal locations: %w", err)
	}
	locations := make([]catalog.Location, 0, len(locationRows))
	for _, row := range locationRows {
		location, err := locationFromRow(row.ID, row.Address, row.Lat, row.Lng, row.OpeningHours, row.Phone,
			row.CityID, row.CitySlug, row.CityName, row.CityLat, row.CityLng)
		if err != nil {
			return catalog.Deal{}, err
		}
		locations = append(locations, location)
	}
	deal, err := detailFromRow(detailRow)
	if err != nil {
		return catalog.Deal{}, err
	}
	deal.DealSummary = summaries[0]
	deal.Locations = locations
	return deal, nil
}

func (r *CatalogRepository) RelatedDealIDs(ctx context.Context, subcategoryID, excludeID uuid.UUID, citySlug string, limit int) ([]uuid.UUID, error) {
	rows, err := r.queries.ListRelatedDealIDs(ctx, sqlc.ListRelatedDealIDsParams{
		SubcategoryID: toPgUUID(subcategoryID),
		ExcludeDealID: toPgUUID(excludeID),
		City:          citySlug,
		PageLimit:     int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("related deal ids: %w", err)
	}
	return toUUIDs(rows), nil
}

func (r *CatalogRepository) DealReviews(ctx context.Context, dealID uuid.UUID, page catalog.Page) ([]catalog.Review, int64, error) {
	rows, err := r.queries.ListPublishedReviewsForDeal(ctx, sqlc.ListPublishedReviewsForDealParams{
		DealID:     toPgUUID(dealID),
		PageOffset: int32(page.Offset()),
		PageLimit:  int32(page.Size),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list reviews: %w", err)
	}
	total, err := r.queries.CountPublishedReviewsForDeal(ctx, toPgUUID(dealID))
	if err != nil {
		return nil, 0, fmt.Errorf("count reviews: %w", err)
	}
	reviews := make([]catalog.Review, 0, len(rows))
	for _, row := range rows {
		reviews = append(reviews, reviewFromRow(row.ID, row.DealID, row.MerchantID, row.UserID, row.VoucherID,
			row.Rating, row.Text, string(row.Locale), string(row.Status), row.CreatedAt,
			row.ReplyID, row.ReplyText, row.ReplyCreatedAt))
	}
	return reviews, total, nil
}

func (r *CatalogRepository) MerchantBySlug(ctx context.Context, slug string) (catalog.Merchant, error) {
	row, err := r.queries.GetActiveMerchantBySlug(ctx, slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return catalog.Merchant{}, catalog.ErrNotFound
	}
	if err != nil {
		return catalog.Merchant{}, fmt.Errorf("get merchant: %w", err)
	}
	description, err := catalog.ParseText(row.Description)
	if err != nil {
		return catalog.Merchant{}, err
	}
	locationRows, err := r.queries.ListMerchantLocationRows(ctx, row.ID)
	if err != nil {
		return catalog.Merchant{}, fmt.Errorf("list merchant locations: %w", err)
	}
	locations := make([]catalog.Location, 0, len(locationRows))
	for _, location := range locationRows {
		item, err := locationFromRow(location.ID, location.Address, location.Lat, location.Lng, location.OpeningHours,
			location.Phone, location.CityID, location.CitySlug, location.CityName, location.CityLat, location.CityLng)
		if err != nil {
			return catalog.Merchant{}, err
		}
		locations = append(locations, item)
	}
	return catalog.Merchant{
		ID:            toUUID(row.ID),
		Slug:          row.Slug,
		LegalName:     row.LegalName,
		DisplayName:   row.DisplayName,
		Description:   description,
		LogoURL:       textPtr(row.LogoUrl),
		CoverURL:      textPtr(row.CoverUrl),
		Phone:         row.Phone,
		Website:       textPtr(row.Website),
		Instagram:     textPtr(row.Instagram),
		Telegram:      textPtr(row.Telegram),
		RatingAvg:     row.RatingAvg,
		RatingCount:   int(row.RatingCount),
		CommissionBPS: int(row.CommissionBps),
		Locations:     locations,
	}, nil
}

func (r *CatalogRepository) MerchantReviews(ctx context.Context, merchantID uuid.UUID, limit int) ([]catalog.Review, error) {
	rows, err := r.queries.ListPublishedReviewsForMerchant(ctx, sqlc.ListPublishedReviewsForMerchantParams{
		MerchantID: toPgUUID(merchantID),
		Limit:      int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("list merchant reviews: %w", err)
	}
	reviews := make([]catalog.Review, 0, len(rows))
	for _, row := range rows {
		reviews = append(reviews, reviewFromRow(row.ID, row.DealID, row.MerchantID, row.UserID, row.VoucherID,
			row.Rating, row.Text, string(row.Locale), string(row.Status), row.CreatedAt,
			row.ReplyID, row.ReplyText, row.ReplyCreatedAt))
	}
	return reviews, nil
}

func (r *CatalogRepository) SuggestDeals(ctx context.Context, query, locale string, limit int) ([]catalog.SuggestionRow, error) {
	rows, err := r.queries.SuggestDeals(ctx, sqlc.SuggestDealsParams{Query: query, PageLimit: int32(limit)})
	if err != nil {
		return nil, fmt.Errorf("suggest deals: %w", err)
	}
	result := make([]catalog.SuggestionRow, 0, len(rows))
	for _, row := range rows {
		title, err := catalog.ParseText(row.Title)
		if err != nil {
			return nil, err
		}
		result = append(result, catalog.SuggestionRow{Type: "deal", Slug: row.Slug, Label: title.Resolve(locale)})
	}
	return result, nil
}

func (r *CatalogRepository) SuggestMerchants(ctx context.Context, query, locale string, limit int) ([]catalog.SuggestionRow, error) {
	rows, err := r.queries.SuggestMerchants(ctx, sqlc.SuggestMerchantsParams{Query: query, PageLimit: int32(limit)})
	if err != nil {
		return nil, fmt.Errorf("suggest merchants: %w", err)
	}
	result := make([]catalog.SuggestionRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, catalog.SuggestionRow{Type: "merchant", Slug: row.Slug, Label: row.DisplayName})
	}
	return result, nil
}

func (r *CatalogRepository) SuggestCategories(ctx context.Context, query, locale string, limit int) ([]catalog.SuggestionRow, error) {
	rows, err := r.queries.SuggestCategories(ctx, sqlc.SuggestCategoriesParams{Query: query, PageLimit: int32(limit)})
	if err != nil {
		return nil, fmt.Errorf("suggest categories: %w", err)
	}
	result := make([]catalog.SuggestionRow, 0, len(rows))
	for _, row := range rows {
		name, err := catalog.ParseText(row.Name)
		if err != nil {
			return nil, err
		}
		result = append(result, catalog.SuggestionRow{Type: "category", Slug: row.Slug, Label: name.Resolve(locale)})
	}
	return result, nil
}

func (r *CatalogRepository) HomepageRows(ctx context.Context) ([]catalog.HomepageRow, error) {
	rows, err := r.queries.ListActiveHomepageSections(ctx)
	if err != nil {
		return nil, fmt.Errorf("list homepage sections: %w", err)
	}
	result := make([]catalog.HomepageRow, 0, len(rows))
	for _, row := range rows {
		title, err := catalog.ParseText(row.Title)
		if err != nil {
			return nil, err
		}
		rule := map[string]any{}
		if len(row.Rule) > 0 {
			if err := json.Unmarshal(row.Rule, &rule); err != nil {
				return nil, fmt.Errorf("decode homepage rule: %w", err)
			}
		}
		result = append(result, catalog.HomepageRow{
			ID:        toUUID(row.ID),
			Key:       row.Key,
			Title:     title,
			Type:      string(row.Type),
			DealIDs:   toUUIDs(row.DealIds),
			Rule:      rule,
			SortOrder: int(row.SortOrder),
		})
	}
	return result, nil
}

func summaryFromRow(row sqlc.ListDealSummaryRowsRow) (catalog.DealSummary, error) {
	title, err := catalog.ParseText(row.Title)
	if err != nil {
		return catalog.DealSummary{}, err
	}
	merchantDescription, err := catalog.ParseText(row.MerchantDescription)
	if err != nil {
		return catalog.DealSummary{}, err
	}
	categoryName, err := catalog.ParseText(row.CategoryName)
	if err != nil {
		return catalog.DealSummary{}, err
	}
	return catalog.DealSummary{
		ID:   toUUID(row.ID),
		Slug: row.Slug,
		Area: row.Area,
		Merchant: catalog.Merchant{
			ID:            toUUID(row.MerchantID),
			Slug:          row.MerchantSlug,
			LegalName:     row.MerchantLegalName,
			DisplayName:   row.MerchantDisplayName,
			Description:   merchantDescription,
			LogoURL:       textPtr(row.MerchantLogoUrl),
			CoverURL:      textPtr(row.MerchantCoverUrl),
			Phone:         row.MerchantPhone,
			Website:       textPtr(row.MerchantWebsite),
			Instagram:     textPtr(row.MerchantInstagram),
			Telegram:      textPtr(row.MerchantTelegram),
			RatingAvg:     row.MerchantRatingAvg,
			RatingCount:   int(row.MerchantRatingCount),
			CommissionBPS: int(row.MerchantCommissionBps),
		},
		Category: catalog.Category{
			ID:            toUUID(row.CategoryID),
			Slug:          row.CategorySlug,
			Name:          categoryName,
			Icon:          row.CategoryIcon,
			Subcategories: []catalog.Subcategory{},
		},
		Title:       title,
		Status:      string(row.Status),
		SoldCount:   int(row.SoldCount),
		RatingAvg:   row.RatingAvg,
		RatingCount: int(row.RatingCount),
		StartsAt:    timePtr(row.StartsAt),
		EndsAt:      row.EndsAt.Time,
	}, nil
}

func detailFromRow(row sqlc.GetDealDetailRowBySlugRow) (catalog.Deal, error) {
	shortDescription, err := catalog.ParseText(row.ShortDescription)
	if err != nil {
		return catalog.Deal{}, err
	}
	description, err := catalog.ParseText(row.Description)
	if err != nil {
		return catalog.Deal{}, err
	}
	terms, err := catalog.ParseText(row.Terms)
	if err != nil {
		return catalog.Deal{}, err
	}
	highlights, err := textList(row.Highlights)
	if err != nil {
		return catalog.Deal{}, err
	}
	included, err := textList(row.WhatsIncluded)
	if err != nil {
		return catalog.Deal{}, err
	}
	faq, err := faqList(row.Faq)
	if err != nil {
		return catalog.Deal{}, err
	}
	var instructions *catalog.Text
	if len(row.RedemptionInstructions) > 0 {
		parsed, err := catalog.ParseText(row.RedemptionInstructions)
		if err != nil {
			return catalog.Deal{}, err
		}
		instructions = &parsed
	}
	subcategoryName, err := catalog.ParseText(row.SubcategoryName)
	if err != nil {
		return catalog.Deal{}, err
	}
	return catalog.Deal{
		Subcategory: catalog.Subcategory{
			ID:   toUUID(row.SubcategoryID),
			Slug: row.SubcategorySlug,
			Name: subcategoryName,
		},
		ShortDescription:       shortDescription,
		Description:            description,
		Highlights:             highlights,
		WhatsIncluded:          included,
		Terms:                  terms,
		FAQ:                    faq,
		RedemptionMethod:       string(row.RedemptionMethod),
		RedemptionInstructions: instructions,
		BookingRequired:        row.BookingRequired,
		VoucherValidDays:       int32PtrToInt(row.VoucherValidDays),
		VoucherValidUntil:      timePtr(row.VoucherValidUntil),
		MaxPerUser:             int(row.MaxPerUser),
	}, nil
}

func optionFromRow(row sqlc.ListActiveDealOptionsForDealsRow) (catalog.Option, error) {
	title, err := catalog.ParseText(row.Title)
	if err != nil {
		return catalog.Option{}, err
	}
	return catalog.Option{
		ID:            toUUID(row.ID),
		Title:         title,
		OriginalPrice: catalog.Money{Amount: row.OriginalPrice},
		DealPrice:     catalog.Money{Amount: row.DealPrice},
		StockTotal:    int(row.StockTotal),
		StockSold:     int(row.StockSold),
		StockReserved: int(row.StockReserved),
		IsActive:      row.IsActive,
	}, nil
}

func mediaFromRow(row sqlc.ListDealMediaForDealsRow) (catalog.Media, error) {
	alt, err := catalog.ParseText(row.Alt)
	if err != nil {
		return catalog.Media{}, err
	}
	return catalog.Media{
		ID:        toUUID(row.ID),
		URL:       row.Url,
		Alt:       alt,
		SortOrder: int(row.SortOrder),
		Width:     int(row.Width),
		Height:    int(row.Height),
		Blurhash:  row.Blurhash,
	}, nil
}

func locationFromRow(id pgtype.UUID, address string, lat, lng float64, hours []byte, phone string,
	cityID pgtype.UUID, citySlug string, cityName []byte, cityLat, cityLng float64) (catalog.Location, error) {
	name, err := catalog.ParseText(cityName)
	if err != nil {
		return catalog.Location{}, err
	}
	openingHours := map[string]any{}
	if len(hours) > 0 {
		if err := json.Unmarshal(hours, &openingHours); err != nil {
			return catalog.Location{}, fmt.Errorf("decode opening hours: %w", err)
		}
	}
	return catalog.Location{
		ID:      toUUID(id),
		Address: address,
		Lat:     lat,
		Lng:     lng,
		Phone:   phone,
		City: catalog.City{
			ID:   toUUID(cityID),
			Slug: citySlug,
			Name: name,
			Lat:  cityLat,
			Lng:  cityLng,
		},
		OpeningHours: openingHours,
	}, nil
}

func reviewFromRow(id, dealID, merchantID, userID, voucherID pgtype.UUID, rating int16, text, locale, status string,
	createdAt pgtype.Timestamptz, replyID pgtype.UUID, replyText pgtype.Text, replyCreatedAt pgtype.Timestamptz) catalog.Review {
	review := catalog.Review{
		ID:         toUUID(id),
		DealID:     toUUID(dealID),
		MerchantID: toUUID(merchantID),
		UserID:     toUUID(userID),
		VoucherID:  toUUID(voucherID),
		Rating:     int(rating),
		Text:       text,
		Locale:     locale,
		Status:     status,
		CreatedAt:  createdAt.Time,
	}
	if replyID.Valid {
		review.Reply = &catalog.ReviewReply{
			ID:        toUUID(replyID),
			Text:      replyText.String,
			CreatedAt: replyCreatedAt.Time,
		}
	}
	return review
}

func textList(raw []byte) ([]catalog.Text, error) {
	list := []catalog.Text{}
	if len(raw) == 0 {
		return list, nil
	}
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, fmt.Errorf("decode localized list: %w", err)
	}
	return list, nil
}

func faqList(raw []byte) ([]catalog.FAQItem, error) {
	type faqRow struct {
		Q catalog.Text `json:"q"`
		A catalog.Text `json:"a"`
	}
	rows := []faqRow{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &rows); err != nil {
			return nil, fmt.Errorf("decode faq: %w", err)
		}
	}
	items := make([]catalog.FAQItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, catalog.FAQItem{Q: row.Q, A: row.A})
	}
	return items, nil
}

func toUUID(id pgtype.UUID) uuid.UUID {
	return uuid.UUID(id.Bytes)
}

func toPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func toPgUUIDs(ids []uuid.UUID) []pgtype.UUID {
	result := make([]pgtype.UUID, 0, len(ids))
	for _, id := range ids {
		result = append(result, toPgUUID(id))
	}
	return result
}

// toUUIDs accepts both id column rows and uuid arrays, which share the pgtype.UUID element type.
func toUUIDs(ids []pgtype.UUID) []uuid.UUID {
	result := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id.Valid {
			result = append(result, toUUID(id))
		}
	}
	return result
}

func optText(value string) pgtype.Text {
	return pgtype.Text{String: value, Valid: value != ""}
}

func optInt64(value *int64) pgtype.Int8 {
	if value == nil {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: *value, Valid: true}
}

func optInt32(value *int) pgtype.Int4 {
	if value == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: int32(*value), Valid: true}
}

func optFloat64(value *float64) pgtype.Float8 {
	if value == nil {
		return pgtype.Float8{}
	}
	return pgtype.Float8{Float64: *value, Valid: true}
}

func textPtr(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	text := value.String
	return &text
}

func timePtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	timestamp := value.Time
	return &timestamp
}

func int32PtrToInt(value pgtype.Int4) *int {
	if !value.Valid {
		return nil
	}
	number := int(value.Int32)
	return &number
}

func nonNil[T any](items []T) []T {
	if items == nil {
		return []T{}
	}
	return items
}

// SuggestAddresses returns merchant addresses that contain the query, so a street name can be suggested.
func (r *CatalogRepository) SuggestAddresses(ctx context.Context, query string, limit int) ([]catalog.SuggestionRow, error) {
	rows, err := r.queries.SuggestAddresses(ctx, sqlc.SuggestAddressesParams{Query: query, PageLimit: int32(limit)})
	if err != nil {
		return nil, fmt.Errorf("suggest addresses: %w", err)
	}
	result := make([]catalog.SuggestionRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, catalog.SuggestionRow{Type: "address", Label: row})
	}
	return result, nil
}
