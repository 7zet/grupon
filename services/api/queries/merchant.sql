-- Merchant registration and lookups. Registration writes are composed in one transaction by the store.

-- name: CreateMerchant :one
INSERT INTO merchants (slug, legal_name, display_name, description, phone)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, slug, legal_name, display_name, phone, status, rating_avg::float8 AS rating_avg,
          rating_count, commission_bps;

-- name: CreateMerchantLocation :one
INSERT INTO merchant_locations (merchant_id, city_id, address, lat, lng, opening_hours, phone)
VALUES ($1, $2, $3, $4, $5, '{}'::jsonb, $6)
RETURNING id;

-- name: ActiveCityExists :one
SELECT EXISTS (SELECT 1 FROM cities WHERE id = $1 AND is_active = true);

-- name: ActiveCategoryExists :one
SELECT EXISTS (SELECT 1 FROM categories WHERE id = $1 AND is_active = true);
