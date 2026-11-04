package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/grupon/grupon/services/api/internal/domain/catalog"
	"github.com/grupon/grupon/services/api/internal/domain/merchant"
	sqlc "github.com/grupon/grupon/services/api/internal/store/sqlc"
)

// MerchantDealRepository persists merchant-owned deals. Each write runs in one transaction with its audit row.
type MerchantDealRepository struct {
	pool    *pgxpool.Pool
	catalog *CatalogRepository
}

// NewMerchantDealRepository binds the deal writes to a pool and the public read model for summaries.
func NewMerchantDealRepository(pool *pgxpool.Pool, catalog *CatalogRepository) *MerchantDealRepository {
	return &MerchantDealRepository{pool: pool, catalog: catalog}
}

var _ merchant.DealRepository = (*MerchantDealRepository)(nil)

// References checks the taxonomy and locations of a draft belong to valid, owned records.
// ReviewStatus reads the business review status for the approval gate.
func (r *MerchantDealRepository) ReviewStatus(ctx context.Context, merchantID uuid.UUID) (string, error) {
	return NewMerchantReviewRepository(r.pool).ReviewStatus(ctx, merchantID)
}

func (r *MerchantDealRepository) References(ctx context.Context, merchantID uuid.UUID, in merchant.DealInput) error {
	queries := sqlc.New(r.pool)
	capabilities, err := queries.CategoryCapabilities(ctx, toPgUUID(in.CategoryID))
	if errors.Is(err, pgx.ErrNoRows) {
		return &merchant.InvalidInputError{Field: "category_id", Code: "category_not_found"}
	}
	if err != nil {
		return fmt.Errorf("check category: %w", err)
	}
	// PRD AT-10 and BR-12: a closed category publishes nothing, and the merchant is told why.
	if !capabilities.DealCreationEnabled {
		return &merchant.InvalidInputError{Field: "category_id", Code: "category_closed"}
	}
	subcategoryCategory, err := queries.SubcategoryCategoryID(ctx, toPgUUID(in.SubcategoryID))
	if errors.Is(err, pgx.ErrNoRows) {
		return &merchant.InvalidInputError{Field: "subcategory_id", Code: "subcategory_not_found"}
	}
	if err != nil {
		return fmt.Errorf("check subcategory: %w", err)
	}
	if toUUID(subcategoryCategory) != in.CategoryID {
		return &merchant.InvalidInputError{Field: "subcategory_id", Code: "subcategory_mismatch"}
	}
	locationIDs := uniqueUUIDs(in.LocationIDs)
	owned, err := queries.CountOwnedLocations(ctx, sqlc.CountOwnedLocationsParams{
		MerchantID:  toPgUUID(merchantID),
		LocationIds: toPgUUIDs(locationIDs),
	})
	if err != nil {
		return fmt.Errorf("check locations: %w", err)
	}
	if int(owned) != len(locationIDs) {
		return &merchant.InvalidInputError{Field: "location_ids", Code: "location_not_found"}
	}
	return nil
}

func (r *MerchantDealRepository) CategoryAllowsDealCreation(ctx context.Context, categoryID uuid.UUID) (bool, error) {
	capabilities, err := sqlc.New(r.pool).CategoryCapabilities(ctx, toPgUUID(categoryID))
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check category: %w", err)
	}
	return capabilities.DealCreationEnabled, nil
}

// Insert stores a new deal with its children and audit row.
func (r *MerchantDealRepository) Insert(ctx context.Context, merchantID uuid.UUID, write merchant.DealWrite, audit merchant.AuditEntry) (uuid.UUID, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("begin deal insert: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	queries := sqlc.New(tx)
	in := write.Input
	id, err := queries.InsertDeal(ctx, sqlc.InsertDealParams{
		Slug:                   in.Slug,
		MerchantID:             toPgUUID(merchantID),
		CategoryID:             toPgUUID(in.CategoryID),
		SubcategoryID:          toPgUUID(in.SubcategoryID),
		Title:                  mustJSON(in.Title),
		ShortDescription:       mustJSON(in.ShortDescription),
		Description:            mustJSON(in.Description),
		Highlights:             mustJSONArray(in.Highlights),
		WhatsIncluded:          mustJSONArray(in.WhatsIncluded),
		Terms:                  mustJSON(in.Terms),
		Faq:                    mustJSONArray(in.FAQ),
		RedemptionMethod:       sqlc.RedemptionMethod(in.RedemptionMethod),
		RedemptionInstructions: mustJSON(in.RedemptionInstructions),
		BookingRequired:        in.BookingRequired,
		Status:                 sqlc.DealStatus(write.Status),
		StartsAt:               pgTime(in.StartsAt),
		EndsAt:                 pgTime(in.EndsAt),
		VoucherValidDays:       pgInt4Ptr(in.VoucherValidDays),
		VoucherValidUntil:      pgTimePtr(in.VoucherValidUntil),
		MaxPerUser:             int32(in.MaxPerUser),
		ReviewFingerprint:      write.Fingerprint,
	})
	if err != nil {
		return uuid.Nil, mapDealWriteError(err)
	}
	dealID := toUUID(id)
	if err := writeDealChildren(ctx, queries, dealID, in); err != nil {
		return uuid.Nil, err
	}
	audit.DealID = dealID
	if err := writeAudit(ctx, queries, audit); err != nil {
		return uuid.Nil, err
	}
	return dealID, tx.Commit(ctx)
}

// Update replaces the draft fields, children and status of an owned deal.
func (r *MerchantDealRepository) Update(ctx context.Context, merchantID, dealID uuid.UUID, write merchant.DealWrite, audit merchant.AuditEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin deal update: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	queries := sqlc.New(tx)
	in := write.Input
	rows, err := queries.UpdateOwnedDeal(ctx, sqlc.UpdateOwnedDealParams{
		ID:                     toPgUUID(dealID),
		MerchantID:             toPgUUID(merchantID),
		Slug:                   in.Slug,
		CategoryID:             toPgUUID(in.CategoryID),
		SubcategoryID:          toPgUUID(in.SubcategoryID),
		Title:                  mustJSON(in.Title),
		ShortDescription:       mustJSON(in.ShortDescription),
		Description:            mustJSON(in.Description),
		Highlights:             mustJSONArray(in.Highlights),
		WhatsIncluded:          mustJSONArray(in.WhatsIncluded),
		Terms:                  mustJSON(in.Terms),
		Faq:                    mustJSONArray(in.FAQ),
		RedemptionMethod:       sqlc.RedemptionMethod(in.RedemptionMethod),
		RedemptionInstructions: mustJSON(in.RedemptionInstructions),
		BookingRequired:        in.BookingRequired,
		Status:                 sqlc.DealStatus(write.Status),
		StartsAt:               pgTime(in.StartsAt),
		EndsAt:                 pgTime(in.EndsAt),
		VoucherValidDays:       pgInt4Ptr(in.VoucherValidDays),
		VoucherValidUntil:      pgTimePtr(in.VoucherValidUntil),
		MaxPerUser:             int32(in.MaxPerUser),
		ReviewFingerprint:      write.Fingerprint,
	})
	if err != nil {
		return mapDealWriteError(err)
	}
	if rows == 0 {
		return merchant.ErrDealNotFound
	}
	if err := writeDealChildren(ctx, queries, dealID, in); err != nil {
		return err
	}
	audit.DealID = dealID
	if err := writeAudit(ctx, queries, audit); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Record reads the workflow state of an owned deal.
func (r *MerchantDealRepository) Record(ctx context.Context, merchantID, dealID uuid.UUID) (merchant.DealRecord, error) {
	row, err := sqlc.New(r.pool).GetOwnedDealRecord(ctx, sqlc.GetOwnedDealRecordParams{
		ID:         toPgUUID(dealID),
		MerchantID: toPgUUID(merchantID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return merchant.DealRecord{}, merchant.ErrDealNotFound
	}
	if err != nil {
		return merchant.DealRecord{}, fmt.Errorf("get deal record: %w", err)
	}
	return merchant.DealRecord{ID: toUUID(row.ID), Status: string(row.Status), Fingerprint: row.ReviewFingerprint, CategoryID: toUUID(row.CategoryID)}, nil
}

// SetStatus moves an owned deal to a new workflow status and audits the change.
func (r *MerchantDealRepository) SetStatus(ctx context.Context, merchantID, dealID uuid.UUID, status string, audit merchant.AuditEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin deal status: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	queries := sqlc.New(tx)
	rows, err := queries.SetOwnedDealStatus(ctx, sqlc.SetOwnedDealStatusParams{
		ID:         toPgUUID(dealID),
		MerchantID: toPgUUID(merchantID),
		Status:     sqlc.DealStatus(status),
	})
	if err != nil {
		return fmt.Errorf("set deal status: %w", err)
	}
	if rows == 0 {
		return merchant.ErrDealNotFound
	}
	audit.DealID = dealID
	if err := writeAudit(ctx, queries, audit); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// SoftDeleteDraft removes a draft owned by the merchant. It reports false when the deal is not a draft.
func (r *MerchantDealRepository) SoftDeleteDraft(ctx context.Context, merchantID, dealID uuid.UUID, audit merchant.AuditEntry) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin deal delete: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	queries := sqlc.New(tx)
	rows, err := queries.SoftDeleteOwnedDraftDeal(ctx, sqlc.SoftDeleteOwnedDraftDealParams{
		ID:         toPgUUID(dealID),
		MerchantID: toPgUUID(merchantID),
	})
	if err != nil {
		return false, fmt.Errorf("soft delete deal: %w", err)
	}
	if rows == 0 {
		return false, nil
	}
	audit.DealID = dealID
	if err := writeAudit(ctx, queries, audit); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

// Detail assembles the full owned deal, including drafts and deals under review.
func (r *MerchantDealRepository) Detail(ctx context.Context, merchantID, dealID uuid.UUID) (catalog.Deal, error) {
	queries := sqlc.New(r.pool)
	row, err := queries.GetOwnedDealDetailRow(ctx, sqlc.GetOwnedDealDetailRowParams{
		ID:         toPgUUID(dealID),
		MerchantID: toPgUUID(merchantID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return catalog.Deal{}, merchant.ErrDealNotFound
	}
	if err != nil {
		return catalog.Deal{}, fmt.Errorf("get owned deal: %w", err)
	}
	deal, err := detailFromRow(detailRowFromOwned(row))
	if err != nil {
		return catalog.Deal{}, err
	}
	summaries, err := r.catalog.DealSummaries(ctx, []uuid.UUID{dealID})
	if err != nil {
		return catalog.Deal{}, err
	}
	if len(summaries) != 1 {
		return catalog.Deal{}, merchant.ErrDealNotFound
	}
	locationRows, err := queries.ListDealLocationRows(ctx, []pgtype.UUID{toPgUUID(dealID)})
	if err != nil {
		return catalog.Deal{}, fmt.Errorf("list deal locations: %w", err)
	}
	locations := make([]catalog.Location, 0, len(locationRows))
	for _, location := range locationRows {
		item, err := locationFromRow(location.ID, location.Address, location.Lat, location.Lng, location.OpeningHours,
			location.Phone, location.CityID, location.CitySlug, location.CityName, location.CityLat, location.CityLng)
		if err != nil {
			return catalog.Deal{}, err
		}
		locations = append(locations, item)
	}
	deal.DealSummary = summaries[0]
	deal.Locations = locations
	return deal, nil
}

// ListIDs returns one page of owned deal ids and the total count for the filter.
func (r *MerchantDealRepository) ListIDs(ctx context.Context, merchantID uuid.UUID, status string, page catalog.Page) ([]uuid.UUID, int64, error) {
	queries := sqlc.New(r.pool)
	filter := sqlc.NullDealStatus{DealStatus: sqlc.DealStatus(status), Valid: status != ""}
	total, err := queries.CountOwnedDeals(ctx, sqlc.CountOwnedDealsParams{MerchantID: toPgUUID(merchantID), Status: filter})
	if err != nil {
		return nil, 0, fmt.Errorf("count owned deals: %w", err)
	}
	ids, err := queries.ListOwnedDealIDs(ctx, sqlc.ListOwnedDealIDsParams{
		MerchantID: toPgUUID(merchantID),
		Status:     filter,
		PageLimit:  int32(page.Size),
		PageOffset: int32(page.Offset()),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list owned deals: %w", err)
	}
	return toUUIDs(ids), total, nil
}

// Summaries returns card views for the given deal ids, in the order requested.
func (r *MerchantDealRepository) Summaries(ctx context.Context, ids []uuid.UUID) ([]catalog.DealSummary, error) {
	return r.catalog.DealSummaries(ctx, ids)
}

func writeDealChildren(ctx context.Context, queries *sqlc.Queries, dealID uuid.UUID, in merchant.DealInput) error {
	keep := make([]uuid.UUID, 0, len(in.Options))
	for _, option := range in.Options {
		id := uuid.New()
		if option.ID != nil {
			id = *option.ID
		}
		keep = append(keep, id)
		if err := queries.UpsertDealOption(ctx, sqlc.UpsertDealOptionParams{
			ID:                 toPgUUID(id),
			DealID:             toPgUUID(dealID),
			Title:              mustJSON(option.Title),
			OriginalPrice:      option.OriginalPrice,
			DealPrice:          option.DealPrice,
			StockTotal:         int32(option.StockTotal),
			SortOrder:          int32(option.SortOrder),
			IsActive:           option.IsActive,
			RedemptionsAllowed: int32(max(1, option.RedemptionsAllowed)),
		}); err != nil {
			return fmt.Errorf("upsert deal option: %w", err)
		}
	}
	if err := queries.DeactivateDealOptionsExcept(ctx, sqlc.DeactivateDealOptionsExceptParams{
		DealID:  toPgUUID(dealID),
		KeepIds: toPgUUIDs(keep),
	}); err != nil {
		return fmt.Errorf("deactivate removed options: %w", err)
	}
	if err := queries.DeleteDealMedia(ctx, toPgUUID(dealID)); err != nil {
		return fmt.Errorf("clear deal media: %w", err)
	}
	for _, item := range in.Media {
		if err := queries.InsertDealMedia(ctx, sqlc.InsertDealMediaParams{
			DealID:    toPgUUID(dealID),
			Url:       item.URL,
			Alt:       mustJSON(item.Alt),
			SortOrder: int32(item.SortOrder),
			Width:     int32(item.Width),
			Height:    int32(item.Height),
			Blurhash:  item.Blurhash,
		}); err != nil {
			return fmt.Errorf("insert deal media: %w", err)
		}
	}
	if err := queries.DeleteDealLocations(ctx, toPgUUID(dealID)); err != nil {
		return fmt.Errorf("clear deal locations: %w", err)
	}
	for _, locationID := range uniqueUUIDs(in.LocationIDs) {
		if err := queries.InsertDealLocation(ctx, sqlc.InsertDealLocationParams{
			DealID:             toPgUUID(dealID),
			MerchantLocationID: toPgUUID(locationID),
		}); err != nil {
			return fmt.Errorf("insert deal location: %w", err)
		}
	}
	return nil
}

func writeAudit(ctx context.Context, queries *sqlc.Queries, audit merchant.AuditEntry) error {
	after, err := json.Marshal(audit.After)
	if err != nil {
		return fmt.Errorf("encode audit: %w", err)
	}
	if err := queries.InsertAuditLog(ctx, sqlc.InsertAuditLogParams{
		ActorUserID: toPgUUID(audit.ActorID),
		ActorRole:   sqlc.AppRole(audit.Role),
		Action:      audit.Action,
		EntityType:  "deal",
		EntityID:    toPgUUID(audit.DealID),
		AfterData:   after,
	}); err != nil {
		return fmt.Errorf("write audit log: %w", err)
	}
	return nil
}

func mapDealWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return &merchant.InvalidInputError{Field: "slug", Code: "slug_taken"}
	}
	return fmt.Errorf("write deal: %w", err)
}

// detailRowFromOwned copies the owned detail row into the shape the public detail mapper reads.
func detailRowFromOwned(row sqlc.GetOwnedDealDetailRowRow) sqlc.GetDealDetailRowBySlugRow {
	return sqlc.GetDealDetailRowBySlugRow{
		ID:                     row.ID,
		Slug:                   row.Slug,
		Title:                  row.Title,
		Status:                 row.Status,
		SoldCount:              row.SoldCount,
		RatingAvg:              row.RatingAvg,
		RatingCount:            row.RatingCount,
		StartsAt:               row.StartsAt,
		EndsAt:                 row.EndsAt,
		ShortDescription:       row.ShortDescription,
		Description:            row.Description,
		Highlights:             row.Highlights,
		WhatsIncluded:          row.WhatsIncluded,
		Terms:                  row.Terms,
		Faq:                    row.Faq,
		RedemptionMethod:       row.RedemptionMethod,
		RedemptionInstructions: row.RedemptionInstructions,
		BookingRequired:        row.BookingRequired,
		VoucherValidDays:       row.VoucherValidDays,
		VoucherValidUntil:      row.VoucherValidUntil,
		MaxPerUser:             row.MaxPerUser,
		MerchantID:             row.MerchantID,
		CategoryID:             row.CategoryID,
		SubcategoryID:          row.SubcategoryID,
		SubcategoryName:        row.SubcategoryName,
		SubcategorySlug:        row.SubcategorySlug,
	}
}

func mustJSON(value any) []byte {
	payload, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Sprintf("encode json: %v", err))
	}
	return payload
}

func pgTime(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: true}
}

func pgTimePtr(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return pgTime(*value)
}

func pgInt4Ptr(value *int) pgtype.Int4 {
	if value == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: int32(*value), Valid: true}
}

func uniqueUUIDs(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]bool, len(ids))
	result := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}
	return result
}

// mustJSONArray encodes a list as a JSON array, so an empty list is stored as [] rather than null.
func mustJSONArray[T any](items []T) []byte {
	if items == nil {
		items = []T{}
	}
	return mustJSON(items)
}
