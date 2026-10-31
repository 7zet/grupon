package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/grupon/grupon/services/api/internal/domain/merchant"
	sqlc "github.com/grupon/grupon/services/api/internal/store/sqlc"
)

// MerchantRepository persists merchant registrations atomically.
type MerchantRepository struct {
	pool *pgxpool.Pool
}

// NewMerchantRepository binds the registration writes to a pool.
func NewMerchantRepository(pool *pgxpool.Pool) *MerchantRepository {
	return &MerchantRepository{pool: pool}
}

var _ merchant.Repository = (*MerchantRepository)(nil)

// RegisterOwned creates the business, its first location and the owner role in one transaction.
// Any failure rolls the whole registration back, so no half-registered business can exist.
func (r *MerchantRepository) RegisterOwned(ctx context.Context, ownerID uuid.UUID, registration merchant.Registration) (merchant.Created, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return merchant.Created{}, fmt.Errorf("begin registration: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	queries := sqlc.New(tx)
	cityOK, err := queries.ActiveCityExists(ctx, toPgUUID(registration.CityID))
	if err != nil {
		return merchant.Created{}, fmt.Errorf("check city: %w", err)
	}
	if !cityOK {
		return merchant.Created{}, merchant.ErrCityNotFound
	}
	categoryOK, err := queries.ActiveCategoryExists(ctx, toPgUUID(registration.CategoryID))
	if err != nil {
		return merchant.Created{}, fmt.Errorf("check category: %w", err)
	}
	if !categoryOK {
		return merchant.Created{}, merchant.ErrCategoryNotFound
	}

	created, err := queries.CreateMerchant(ctx, sqlc.CreateMerchantParams{
		Slug:        registration.Slug,
		LegalName:   registration.LegalName,
		DisplayName: registration.DisplayName,
		Description: []byte(`{"uz":"","ru":"","en":""}`),
		Phone:       registration.Phone,
	})
	if err != nil {
		return merchant.Created{}, fmt.Errorf("create merchant: %w", err)
	}

	lat, err := numericFromFloat(registration.Lat)
	if err != nil {
		return merchant.Created{}, err
	}
	lng, err := numericFromFloat(registration.Lng)
	if err != nil {
		return merchant.Created{}, err
	}
	if _, err := queries.CreateMerchantLocation(ctx, sqlc.CreateMerchantLocationParams{
		MerchantID: created.ID,
		CityID:     toPgUUID(registration.CityID),
		Address:    registration.Address,
		Lat:        lat,
		Lng:        lng,
		Phone:      registration.Phone,
	}); err != nil {
		return merchant.Created{}, fmt.Errorf("create location: %w", err)
	}

	if err := queries.CreateRoleAssignment(ctx, sqlc.CreateRoleAssignmentParams{
		UserID:     toPgUUID(ownerID),
		Role:       sqlc.AppRole("merchant_owner"),
		MerchantID: created.ID,
	}); err != nil {
		return merchant.Created{}, fmt.Errorf("assign owner role: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return merchant.Created{}, fmt.Errorf("commit registration: %w", err)
	}
	return merchant.Created{
		ID:          toUUID(created.ID),
		Slug:        created.Slug,
		DisplayName: created.DisplayName,
		LegalName:   created.LegalName,
		Phone:       created.Phone,
		Status:      string(created.Status),
	}, nil
}

func numericFromFloat(value float64) (pgtype.Numeric, error) {
	var number pgtype.Numeric
	if err := number.Scan(fmt.Sprintf("%.6f", value)); err != nil {
		return pgtype.Numeric{}, fmt.Errorf("encode coordinate: %w", err)
	}
	return number, nil
}
