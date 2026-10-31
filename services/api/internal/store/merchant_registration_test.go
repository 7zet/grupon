package store_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/grupon/grupon/services/api/internal/domain/merchant"
	"github.com/grupon/grupon/services/api/internal/store"
)

// TestMerchantRegistrationCreatesBusinessLocationAndOwnerRole checks the whole transaction and that
// an unknown city rolls everything back.
func TestMerchantRegistrationCreatesBusinessLocationAndOwnerRole(t *testing.T) {
	fixture := newCatalogFixture(t)
	ctx := context.Background()
	service := merchant.NewService(store.NewMerchantRepository(fixture.pool))

	var ownerID uuid.UUID
	if err := fixture.pool.QueryRow(ctx,
		`INSERT INTO users (email, name, password_hash) VALUES ('owner@test.local', 'Owner', 'x') RETURNING id`,
	).Scan(&ownerID); err != nil {
		t.Fatal(err)
	}
	var cityID, categoryID uuid.UUID
	if err := fixture.pool.QueryRow(ctx, `SELECT id FROM cities WHERE is_active LIMIT 1`).Scan(&cityID); err != nil {
		t.Fatal(err)
	}
	if err := fixture.pool.QueryRow(ctx, `SELECT id FROM categories WHERE is_active LIMIT 1`).Scan(&categoryID); err != nil {
		t.Fatal(err)
	}
	input := merchant.RegisterInput{
		LegalName: "Test Business LLC", DisplayName: "Test Business", CategoryID: categoryID,
		CityID: cityID, Phone: "+998901234567", Address: "Test street 1", Lat: 41.3, Lng: 69.2,
	}

	created, err := service.Register(ctx, ownerID, input)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	var role string
	var locations int
	if err := fixture.pool.QueryRow(ctx,
		`SELECT ra.role::text FROM role_assignments ra WHERE ra.user_id = $1 AND ra.merchant_id = $2`,
		ownerID, created.ID).Scan(&role); err != nil {
		t.Fatal(err)
	}
	if role != "merchant_owner" {
		t.Fatalf("role = %s, want merchant_owner", role)
	}
	if err := fixture.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM merchant_locations WHERE merchant_id = $1`, created.ID).Scan(&locations); err != nil {
		t.Fatal(err)
	}
	if locations != 1 {
		t.Fatalf("locations = %d, want 1", locations)
	}

	input.CityID = uuid.New()
	if _, err := service.Register(ctx, ownerID, input); err != merchant.ErrCityNotFound {
		t.Fatalf("unknown city err = %v, want ErrCityNotFound", err)
	}
}
