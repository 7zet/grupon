-- name: ListActiveCities :many
SELECT * FROM cities WHERE is_active = true ORDER BY sort_order, slug;

-- name: ListActiveCategories :many
SELECT * FROM categories WHERE is_active = true ORDER BY sort_order, slug;

-- name: ListSubcategoriesByCategory :many
SELECT * FROM subcategories WHERE category_id = $1 ORDER BY sort_order, slug;

-- name: GetActiveDealBySlug :one
SELECT * FROM deals WHERE slug = $1 AND deleted_at IS NULL AND status IN ('scheduled', 'active', 'sold_out', 'expired');

-- name: ListDealOptions :many
SELECT * FROM deal_options WHERE deal_id = $1 ORDER BY sort_order, id;

-- name: ListDealMedia :many
SELECT * FROM deal_media WHERE deal_id = $1 ORDER BY sort_order, id;

-- name: ListDealLocations :many
SELECT ml.* FROM merchant_locations ml JOIN deal_locations dl ON dl.merchant_location_id = ml.id WHERE dl.deal_id = $1 ORDER BY ml.id;

-- name: ListActivePromoCodes :many
SELECT * FROM promo_codes WHERE is_active = true AND starts_at <= now() AND ends_at > now() ORDER BY is_verified DESC, ends_at LIMIT $1 OFFSET $2;

-- name: LatestExchangeRate :one
SELECT * FROM exchange_rates WHERE base = $1 AND quote = $2 ORDER BY effective_at DESC LIMIT 1;
