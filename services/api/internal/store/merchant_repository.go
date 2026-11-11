package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dealo/dealo/services/api/internal/domain/merchant"
	sqlc "github.com/dealo/dealo/services/api/internal/store/sqlc"
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
		Slug:               registration.Slug,
		LegalName:          registration.LegalName,
		DisplayName:        registration.DisplayName,
		Description:        []byte(`{"uz":"","ru":"","en":""}`),
		Phone:              registration.Phone,
		Stir:               optText(registration.STIR),
		OrgForm:            optText(registration.OrgForm),
		TaxRegime:          optText(registration.TaxRegime),
		BankMfo:            optText(registration.BankMFO),
		BankAccount:        optText(registration.BankAccount),
		ResponsibleName:    optText(registration.ResponsibleName),
		ContractVersion:    optText(registration.ContractVersion),
		ContractAcceptedAt: pgtype.Timestamptz{Time: registration.ContractAcceptedAt, Valid: true},
		ContractAcceptedIp: optText(registration.ContractIP),
	})
	if err != nil {
		if isUniqueViolation(err) && strings.Contains(err.Error(), "merchants_stir_key") {
			return merchant.Created{}, &merchant.InvalidInputError{Field: "stir", Code: "stir_taken"}
		}
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
		ID:           toUUID(created.ID),
		Slug:         created.Slug,
		DisplayName:  created.DisplayName,
		LegalName:    created.LegalName,
		Phone:        created.Phone,
		Status:       string(created.Status),
		ReviewStatus: created.ReviewStatus,
	}, nil
}

// SubmitOwned sends the owner's draft business to review. The status is checked first so that a
// missing business and an already submitted one get different errors.
func (r *MerchantRepository) SubmitOwned(ctx context.Context, ownerID uuid.UUID, at time.Time) (merchant.Created, error) {
	queries := sqlc.New(r.pool)
	owned, err := queries.OwnedMerchantReviewStatus(ctx, toPgUUID(ownerID))
	if errors.Is(err, pgx.ErrNoRows) {
		return merchant.Created{}, merchant.ErrNoBusiness
	}
	if err != nil {
		return merchant.Created{}, fmt.Errorf("load owned business: %w", err)
	}
	if owned.ReviewStatus != "draft" {
		return merchant.Created{}, merchant.ErrNotDraft
	}
	kinds, err := queries.CountMerchantDocumentKinds(ctx, owned.ID)
	if err != nil {
		return merchant.Created{}, fmt.Errorf("count documents: %w", err)
	}
	if kinds < 2 {
		return merchant.Created{}, merchant.ErrDocumentsRequired
	}
	row, err := queries.SubmitMerchantForReview(ctx, sqlc.SubmitMerchantForReviewParams{
		ID: owned.ID,
		At: pgtype.Timestamptz{Time: at, Valid: true},
	})
	if errors.Is(err, pgx.ErrNoRows) {
		// Another request submitted it between the read and the write.
		return merchant.Created{}, merchant.ErrNotDraft
	}
	if err != nil {
		return merchant.Created{}, fmt.Errorf("submit business: %w", err)
	}
	return merchant.Created{
		ID:           toUUID(row.ID),
		Slug:         row.Slug,
		DisplayName:  row.DisplayName,
		LegalName:    row.LegalName,
		Phone:        row.Phone,
		Status:       string(row.Status),
		ReviewStatus: row.ReviewStatus,
	}, nil
}

func numericFromFloat(value float64) (pgtype.Numeric, error) {
	var number pgtype.Numeric
	if err := number.Scan(fmt.Sprintf("%.6f", value)); err != nil {
		return pgtype.Numeric{}, fmt.Errorf("encode coordinate: %w", err)
	}
	return number, nil
}
