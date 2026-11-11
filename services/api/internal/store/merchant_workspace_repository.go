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
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dealo/dealo/services/api/internal/domain/catalog"
	"github.com/dealo/dealo/services/api/internal/domain/merchant"
	sqlc "github.com/dealo/dealo/services/api/internal/store/sqlc"
)

// MerchantWorkspaceRepository implements the merchant workspace port. Each query is scoped to one
// merchant id, so the adapter cannot read or write another business's rows.
type MerchantWorkspaceRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
	catalog *CatalogRepository
}

// NewMerchantWorkspaceRepository binds the workspace queries to a pool.
func NewMerchantWorkspaceRepository(pool *pgxpool.Pool) *MerchantWorkspaceRepository {
	return &MerchantWorkspaceRepository{pool: pool, queries: sqlc.New(pool), catalog: NewCatalogRepository(pool)}
}

var _ merchant.WorkspaceRepository = (*MerchantWorkspaceRepository)(nil)

func (r *MerchantWorkspaceRepository) RoleFor(ctx context.Context, userID uuid.UUID) (uuid.UUID, string, error) {
	row, err := r.queries.MerchantRoleForUser(ctx, toPgUUID(userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, "", merchant.ErrNoMerchant
	}
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("merchant role: %w", err)
	}
	return toUUID(row.MerchantID), row.Role, nil
}

func (r *MerchantWorkspaceRepository) Business(ctx context.Context, merchantID uuid.UUID) (merchant.Business, error) {
	row, err := r.queries.GetMerchantWorkspace(ctx, toPgUUID(merchantID))
	if errors.Is(err, pgx.ErrNoRows) {
		return merchant.Business{}, merchant.ErrNotFound
	}
	if err != nil {
		return merchant.Business{}, fmt.Errorf("load business: %w", err)
	}
	business, err := businessFromRow(row.ID, row.Slug, row.LegalName, row.DisplayName, row.Description, row.LogoUrl,
		row.CoverUrl, row.Phone, row.Website, row.Instagram, row.Telegram, string(row.Status), row.RatingAvg, row.RatingCount)
	if err != nil {
		return merchant.Business{}, err
	}
	business.ReviewStatus = row.ReviewStatus
	return business, nil
}

func (r *MerchantWorkspaceRepository) UpdateBusiness(ctx context.Context, merchantID uuid.UUID, business merchant.Business) (merchant.Business, error) {
	description, err := json.Marshal(business.Description)
	if err != nil {
		return merchant.Business{}, fmt.Errorf("encode description: %w", err)
	}
	row, err := r.queries.UpdateMerchantWorkspace(ctx, sqlc.UpdateMerchantWorkspaceParams{
		ID:          toPgUUID(merchantID),
		DisplayName: business.DisplayName,
		Description: description,
		LogoUrl:     optTextPtr(business.LogoURL),
		CoverUrl:    optTextPtr(business.CoverURL),
		Phone:       business.Phone,
		Website:     optTextPtr(business.Website),
		Instagram:   optTextPtr(business.Instagram),
		Telegram:    optTextPtr(business.Telegram),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return merchant.Business{}, merchant.ErrNotFound
	}
	if err != nil {
		return merchant.Business{}, fmt.Errorf("update business: %w", err)
	}
	return businessFromRow(row.ID, row.Slug, row.LegalName, row.DisplayName, row.Description, row.LogoUrl,
		row.CoverUrl, row.Phone, row.Website, row.Instagram, row.Telegram, string(row.Status), row.RatingAvg, row.RatingCount)
}

func (r *MerchantWorkspaceRepository) Locations(ctx context.Context, merchantID uuid.UUID) ([]merchant.Location, error) {
	rows, err := r.queries.ListMerchantLocationRows(ctx, toPgUUID(merchantID))
	if err != nil {
		return nil, fmt.Errorf("list locations: %w", err)
	}
	locations := make([]merchant.Location, 0, len(rows))
	for _, row := range rows {
		location, err := workspaceLocation(row.ID, row.Address, row.Lat, row.Lng, row.OpeningHours, row.Phone,
			row.CityID, row.CitySlug, row.CityName, row.CityLat, row.CityLng)
		if err != nil {
			return nil, err
		}
		locations = append(locations, location)
	}
	return locations, nil
}

func (r *MerchantWorkspaceRepository) CreateLocation(ctx context.Context, merchantID uuid.UUID, input merchant.LocationInput) (merchant.Location, error) {
	cityOK, err := r.queries.ActiveCityExists(ctx, toPgUUID(input.CityID))
	if err != nil {
		return merchant.Location{}, fmt.Errorf("check city: %w", err)
	}
	if !cityOK {
		return merchant.Location{}, merchant.ErrCityNotFound
	}
	lat, err := numericFromFloat(input.Lat)
	if err != nil {
		return merchant.Location{}, err
	}
	lng, err := numericFromFloat(input.Lng)
	if err != nil {
		return merchant.Location{}, err
	}
	hours, err := json.Marshal(nonNilMap(input.OpeningHours))
	if err != nil {
		return merchant.Location{}, fmt.Errorf("encode hours: %w", err)
	}
	created, err := r.queries.CreateMerchantLocationWithHours(ctx, sqlc.CreateMerchantLocationWithHoursParams{
		MerchantID:   toPgUUID(merchantID),
		CityID:       toPgUUID(input.CityID),
		Address:      input.Address,
		Lat:          lat,
		Lng:          lng,
		OpeningHours: hours,
		Phone:        input.Phone,
	})
	if err != nil {
		return merchant.Location{}, fmt.Errorf("create location: %w", err)
	}
	locations, err := r.Locations(ctx, merchantID)
	if err != nil {
		return merchant.Location{}, err
	}
	for _, location := range locations {
		if location.ID == toUUID(created) {
			return location, nil
		}
	}
	return merchant.Location{}, merchant.ErrNotFound
}

func (r *MerchantWorkspaceRepository) UpdateLocation(ctx context.Context, merchantID, locationID uuid.UUID, input merchant.LocationInput) (bool, error) {
	cityOK, err := r.queries.ActiveCityExists(ctx, toPgUUID(input.CityID))
	if err != nil {
		return false, fmt.Errorf("check city: %w", err)
	}
	if !cityOK {
		return false, merchant.ErrCityNotFound
	}
	lat, err := numericFromFloat(input.Lat)
	if err != nil {
		return false, err
	}
	lng, err := numericFromFloat(input.Lng)
	if err != nil {
		return false, err
	}
	hours, err := json.Marshal(nonNilMap(input.OpeningHours))
	if err != nil {
		return false, fmt.Errorf("encode hours: %w", err)
	}
	rows, err := r.queries.UpdateMerchantLocationRow(ctx, sqlc.UpdateMerchantLocationRowParams{
		ID:           toPgUUID(locationID),
		MerchantID:   toPgUUID(merchantID),
		CityID:       toPgUUID(input.CityID),
		Address:      input.Address,
		Lat:          lat,
		Lng:          lng,
		OpeningHours: hours,
		Phone:        input.Phone,
	})
	if err != nil {
		return false, fmt.Errorf("update location: %w", err)
	}
	return rows > 0, nil
}

// DeleteLocation reports deleted=false when the row is missing or used by a live deal; the service
// distinguishes the two by re-reading ownership.
func (r *MerchantWorkspaceRepository) DeleteLocation(ctx context.Context, merchantID, locationID uuid.UUID) (bool, error) {
	rows, err := r.queries.DeleteMerchantLocationRow(ctx, sqlc.DeleteMerchantLocationRowParams{
		ID:         toPgUUID(locationID),
		MerchantID: toPgUUID(merchantID),
	})
	if err != nil {
		return false, fmt.Errorf("delete location: %w", err)
	}
	return rows > 0, nil
}

func (r *MerchantWorkspaceRepository) Sales(ctx context.Context, merchantID uuid.UUID, from, to time.Time) (int64, error) {
	total, err := r.queries.DashboardSales(ctx, sqlc.DashboardSalesParams{
		MerchantID: toPgUUID(merchantID), PaidAt: timestamptz(from), PaidAt_2: timestamptz(to),
	})
	if err != nil {
		return 0, fmt.Errorf("sales: %w", err)
	}
	return total, nil
}

func (r *MerchantWorkspaceRepository) VouchersSold(ctx context.Context, merchantID uuid.UUID, from, to time.Time) (int64, error) {
	total, err := r.queries.DashboardVouchersSold(ctx, sqlc.DashboardVouchersSoldParams{
		MerchantID: toPgUUID(merchantID), CreatedAt: timestamptz(from), CreatedAt_2: timestamptz(to),
	})
	if err != nil {
		return 0, fmt.Errorf("vouchers sold: %w", err)
	}
	return total, nil
}

func (r *MerchantWorkspaceRepository) VouchersRedeemed(ctx context.Context, merchantID uuid.UUID, from, to time.Time) (int64, error) {
	total, err := r.queries.DashboardVouchersRedeemed(ctx, sqlc.DashboardVouchersRedeemedParams{
		MerchantID: toPgUUID(merchantID), RedeemedAt: timestamptz(from), RedeemedAt_2: timestamptz(to),
	})
	if err != nil {
		return 0, fmt.Errorf("vouchers redeemed: %w", err)
	}
	return total, nil
}

func (r *MerchantWorkspaceRepository) ActiveDeals(ctx context.Context, merchantID uuid.UUID) (int64, error) {
	total, err := r.queries.DashboardActiveDeals(ctx, toPgUUID(merchantID))
	if err != nil {
		return 0, fmt.Errorf("active deals: %w", err)
	}
	return total, nil
}

func (r *MerchantWorkspaceRepository) SalesByDay(ctx context.Context, merchantID uuid.UUID, from, to time.Time) ([]merchant.ChartPoint, error) {
	rows, err := r.queries.DashboardSalesChart(ctx, sqlc.DashboardSalesChartParams{
		MerchantID: toPgUUID(merchantID), PaidAt: timestamptz(from), PaidAt_2: timestamptz(to),
	})
	if err != nil {
		return nil, fmt.Errorf("sales chart: %w", err)
	}
	points := make([]merchant.ChartPoint, 0, len(rows))
	for _, row := range rows {
		points = append(points, merchant.ChartPoint{Day: row.Day.Time, Value: row.Total})
	}
	return points, nil
}

func (r *MerchantWorkspaceRepository) PendingDealIDs(ctx context.Context, merchantID uuid.UUID, limit int) ([]uuid.UUID, error) {
	rows, err := r.queries.DashboardPendingDealIDs(ctx, sqlc.DashboardPendingDealIDsParams{
		MerchantID: toPgUUID(merchantID), Limit: int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("pending deals: %w", err)
	}
	return toUUIDs(rows), nil
}

func (r *MerchantWorkspaceRepository) DealSummaries(ctx context.Context, ids []uuid.UUID) ([]catalog.DealSummary, error) {
	return r.catalog.DealSummaries(ctx, ids)
}

func businessFromRow(id pgtype.UUID, slug, legal, display string, description []byte, logo, cover pgtype.Text,
	phone string, website, instagram, telegram pgtype.Text, status string, rating float64, count int32) (merchant.Business, error) {
	text, err := catalog.ParseText(description)
	if err != nil {
		return merchant.Business{}, err
	}
	return merchant.Business{
		ID:          toUUID(id),
		Slug:        slug,
		LegalName:   legal,
		DisplayName: display,
		Description: text,
		LogoURL:     textPtr(logo),
		CoverURL:    textPtr(cover),
		Phone:       phone,
		Website:     textPtr(website),
		Instagram:   textPtr(instagram),
		Telegram:    textPtr(telegram),
		Status:      status,
		RatingAvg:   rating,
		RatingCount: int(count),
	}, nil
}

func workspaceLocation(id pgtype.UUID, address string, lat, lng float64, hours []byte, phone string,
	cityID pgtype.UUID, citySlug string, cityName []byte, cityLat, cityLng float64) (merchant.Location, error) {
	name, err := catalog.ParseText(cityName)
	if err != nil {
		return merchant.Location{}, err
	}
	opening := map[string]any{}
	if len(hours) > 0 {
		if err := json.Unmarshal(hours, &opening); err != nil {
			return merchant.Location{}, fmt.Errorf("decode hours: %w", err)
		}
	}
	return merchant.Location{
		ID:           toUUID(id),
		City:         catalog.City{ID: toUUID(cityID), Slug: citySlug, Name: name, Lat: cityLat, Lng: cityLng},
		Address:      address,
		Lat:          lat,
		Lng:          lng,
		OpeningHours: opening,
		Phone:        phone,
	}, nil
}

func timestamptz(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: true}
}

func nonNilMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value
}
