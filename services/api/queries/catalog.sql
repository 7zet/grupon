-- Public catalog read model. Every query here is read-only and filters to what the public may see.

-- name: ListActiveCities :many
SELECT id, slug, name, lat::float8 AS lat, lng::float8 AS lng
FROM cities
WHERE is_active = true
ORDER BY sort_order, slug;

-- name: ListActiveCategories :many
SELECT id, slug, name, icon
FROM categories
WHERE is_active = true
ORDER BY sort_order, slug;

-- name: ListSubcategories :many
SELECT id, category_id, slug, name
FROM subcategories
ORDER BY sort_order, slug;

-- name: GetActiveMerchantBySlug :one
SELECT id, slug, legal_name, display_name, description, logo_url, cover_url, phone, website,
       instagram, telegram, status, rating_avg::float8 AS rating_avg, rating_count, commission_bps
FROM merchants
WHERE slug = $1 AND deleted_at IS NULL AND status = 'active';

-- name: ListMerchantLocationRows :many
SELECT ml.id, ml.address, ml.lat::float8 AS lat, ml.lng::float8 AS lng, ml.opening_hours, ml.phone,
       ci.id AS city_id, ci.slug AS city_slug, ci.name AS city_name, ci.lat::float8 AS city_lat,
       ci.lng::float8 AS city_lng
FROM merchant_locations ml
JOIN cities ci ON ci.id = ml.city_id
WHERE ml.merchant_id = $1
ORDER BY ml.created_at, ml.id;

-- name: ListDealLocationRows :many
SELECT ml.id, ml.address, ml.lat::float8 AS lat, ml.lng::float8 AS lng, ml.opening_hours, ml.phone,
       ci.id AS city_id, ci.slug AS city_slug, ci.name AS city_name, ci.lat::float8 AS city_lat,
       ci.lng::float8 AS city_lng, dl.deal_id
FROM deal_locations dl
JOIN merchant_locations ml ON ml.id = dl.merchant_location_id
JOIN cities ci ON ci.id = ml.city_id
WHERE dl.deal_id = ANY(sqlc.arg(deal_ids)::uuid[])
ORDER BY dl.deal_id, ml.created_at, ml.id;

-- name: GetDealDetailRowBySlug :one
SELECT d.id, d.slug, d.title, d.status, d.sold_count, d.rating_avg::float8 AS rating_avg,
       d.rating_count, d.starts_at, d.ends_at, d.short_description, d.description, d.highlights,
       d.whats_included, d.terms, d.faq, d.redemption_method, d.redemption_instructions,
       d.booking_required, d.voucher_valid_days, d.voucher_valid_until, d.max_per_user,
       d.merchant_id, d.category_id, d.subcategory_id, sc.name AS subcategory_name, sc.slug AS subcategory_slug
FROM deals d
JOIN subcategories sc ON sc.id = d.subcategory_id
WHERE d.slug = $1
  AND d.deleted_at IS NULL
  AND d.status IN ('scheduled', 'active', 'sold_out', 'expired');

-- name: ListDealSummaryRows :many
SELECT d.id, d.slug, d.title, d.status, d.sold_count, d.rating_avg::float8 AS rating_avg,
       d.rating_count, d.starts_at, d.ends_at,
       m.id AS merchant_id, m.slug AS merchant_slug, m.legal_name AS merchant_legal_name,
       m.display_name AS merchant_display_name, m.description AS merchant_description,
       m.logo_url AS merchant_logo_url, m.cover_url AS merchant_cover_url, m.phone AS merchant_phone,
       m.website AS merchant_website, m.instagram AS merchant_instagram, m.telegram AS merchant_telegram,
       m.status AS merchant_status, m.rating_avg::float8 AS merchant_rating_avg,
       m.rating_count AS merchant_rating_count, m.commission_bps AS merchant_commission_bps,
       c.id AS category_id, c.slug AS category_slug, c.name AS category_name, c.icon AS category_icon
FROM deals d
JOIN merchants m ON m.id = d.merchant_id AND m.status = 'active' AND m.deleted_at IS NULL
JOIN categories c ON c.id = d.category_id
WHERE d.id = ANY(sqlc.arg(deal_ids)::uuid[])
  AND d.deleted_at IS NULL;

-- name: ListActiveDealOptionsForDeals :many
SELECT id, deal_id, title, original_price, deal_price, stock_total, stock_sold, stock_reserved, is_active
FROM deal_options
WHERE deal_id = ANY(sqlc.arg(deal_ids)::uuid[]) AND is_active = true
ORDER BY deal_id, sort_order, id;

-- name: ListDealMediaForDeals :many
SELECT id, deal_id, url, alt, sort_order, width, height, blurhash
FROM deal_media
WHERE deal_id = ANY(sqlc.arg(deal_ids)::uuid[])
ORDER BY deal_id, sort_order, id;

-- name: SearchPublishedDealIDs :many
WITH candidates AS (
  SELECT d.id, d.sold_count, d.starts_at, d.ends_at, d.featured_rank,
         pr.min_price, pr.max_discount
  FROM deals d
  JOIN merchants m ON m.id = d.merchant_id AND m.status = 'active' AND m.deleted_at IS NULL
  JOIN categories cat ON cat.id = d.category_id AND cat.is_active = true
  JOIN subcategories sub ON sub.id = d.subcategory_id
  CROSS JOIN LATERAL (
    SELECT MIN(o.deal_price) AS min_price,
           MAX(((o.original_price - o.deal_price) * 200 + o.original_price) / (2 * o.original_price)) AS max_discount
    FROM deal_options o
    WHERE o.deal_id = d.id AND o.is_active = true
  ) pr
  WHERE d.deleted_at IS NULL
    AND d.status IN ('active', 'sold_out')
    AND d.ends_at > now()
    AND pr.min_price IS NOT NULL
    AND (sqlc.narg('merchant')::text IS NULL OR m.slug = sqlc.narg('merchant')::text)
    AND (sqlc.narg('category')::text IS NULL OR cat.slug = sqlc.narg('category')::text)
    AND (sqlc.narg('subcategory')::text IS NULL OR sub.slug = sqlc.narg('subcategory')::text)
    AND (sqlc.narg('city')::text IS NULL OR EXISTS (
          SELECT 1
          FROM deal_locations dl
          JOIN merchant_locations ml ON ml.id = dl.merchant_location_id
          JOIN cities ci ON ci.id = ml.city_id
          WHERE dl.deal_id = d.id AND ci.slug = sqlc.narg('city')::text
        ))
    AND (sqlc.narg('query')::text IS NULL OR (
          d.title_uz_search @@ plainto_tsquery('simple', immutable_unaccent(sqlc.narg('query')::text))
          OR d.title_ru_search @@ plainto_tsquery('russian', immutable_unaccent(sqlc.narg('query')::text))
          OR d.title_en_search @@ plainto_tsquery('english', immutable_unaccent(sqlc.narg('query')::text))
          OR word_similarity(immutable_unaccent(sqlc.narg('query')::text),
               immutable_unaccent(concat_ws(' ', d.title->>'uz', d.title->>'ru', d.title->>'en'))) >= 0.35
          OR word_similarity(immutable_unaccent(sqlc.narg('query')::text),
               immutable_unaccent(m.display_name)) >= 0.35
        ))
    AND (sqlc.narg('min_price')::bigint IS NULL OR pr.min_price >= sqlc.narg('min_price')::bigint)
    AND (sqlc.narg('max_price')::bigint IS NULL OR pr.min_price <= sqlc.narg('max_price')::bigint)
    AND (sqlc.narg('min_rating')::float8 IS NULL OR d.rating_avg >= sqlc.narg('min_rating')::float8)
    AND (sqlc.narg('min_discount')::integer IS NULL OR pr.max_discount >= sqlc.narg('min_discount')::integer)
    AND (sqlc.arg('ending_soon')::boolean = false OR d.ends_at <= now() + interval '72 hours')
)
SELECT id
FROM candidates
ORDER BY
  CASE WHEN sqlc.arg('sort')::text = 'price_asc' THEN min_price END ASC NULLS LAST,
  CASE WHEN sqlc.arg('sort')::text = 'price_desc' THEN min_price END DESC NULLS LAST,
  CASE WHEN sqlc.arg('sort')::text = 'biggest_discount' THEN max_discount END DESC NULLS LAST,
  CASE WHEN sqlc.arg('sort')::text = 'popular' THEN sold_count END DESC,
  CASE WHEN sqlc.arg('sort')::text = 'newest' THEN starts_at END DESC NULLS LAST,
  CASE WHEN sqlc.arg('sort')::text = 'ending_soon' THEN ends_at END ASC,
  featured_rank ASC NULLS LAST,
  sold_count DESC,
  ends_at ASC,
  id ASC
LIMIT sqlc.arg('page_limit') OFFSET sqlc.arg('page_offset');

-- name: CountPublishedDeals :one
SELECT COUNT(*)
FROM deals d
JOIN merchants m ON m.id = d.merchant_id AND m.status = 'active' AND m.deleted_at IS NULL
JOIN categories cat ON cat.id = d.category_id AND cat.is_active = true
JOIN subcategories sub ON sub.id = d.subcategory_id
CROSS JOIN LATERAL (
  SELECT MIN(o.deal_price) AS min_price,
         MAX(((o.original_price - o.deal_price) * 200 + o.original_price) / (2 * o.original_price)) AS max_discount
  FROM deal_options o
  WHERE o.deal_id = d.id AND o.is_active = true
) pr
WHERE d.deleted_at IS NULL
  AND d.status IN ('active', 'sold_out')
  AND d.ends_at > now()
  AND pr.min_price IS NOT NULL
  AND (sqlc.narg('merchant')::text IS NULL OR m.slug = sqlc.narg('merchant')::text)
  AND (sqlc.narg('category')::text IS NULL OR cat.slug = sqlc.narg('category')::text)
  AND (sqlc.narg('subcategory')::text IS NULL OR sub.slug = sqlc.narg('subcategory')::text)
  AND (sqlc.narg('city')::text IS NULL OR EXISTS (
        SELECT 1
        FROM deal_locations dl
        JOIN merchant_locations ml ON ml.id = dl.merchant_location_id
        JOIN cities ci ON ci.id = ml.city_id
        WHERE dl.deal_id = d.id AND ci.slug = sqlc.narg('city')::text
      ))
  AND (sqlc.narg('query')::text IS NULL OR (
        d.title_uz_search @@ plainto_tsquery('simple', immutable_unaccent(sqlc.narg('query')::text))
        OR d.title_ru_search @@ plainto_tsquery('russian', immutable_unaccent(sqlc.narg('query')::text))
        OR d.title_en_search @@ plainto_tsquery('english', immutable_unaccent(sqlc.narg('query')::text))
        OR word_similarity(immutable_unaccent(sqlc.narg('query')::text),
             immutable_unaccent(concat_ws(' ', d.title->>'uz', d.title->>'ru', d.title->>'en'))) >= 0.35
        OR word_similarity(immutable_unaccent(sqlc.narg('query')::text),
             immutable_unaccent(m.display_name)) >= 0.35
      ))
  AND (sqlc.narg('min_price')::bigint IS NULL OR pr.min_price >= sqlc.narg('min_price')::bigint)
  AND (sqlc.narg('max_price')::bigint IS NULL OR pr.min_price <= sqlc.narg('max_price')::bigint)
  AND (sqlc.narg('min_rating')::float8 IS NULL OR d.rating_avg >= sqlc.narg('min_rating')::float8)
  AND (sqlc.narg('min_discount')::integer IS NULL OR pr.max_discount >= sqlc.narg('min_discount')::integer)
  AND (sqlc.arg('ending_soon')::boolean = false OR d.ends_at <= now() + interval '72 hours');

-- name: ListRelatedDealIDs :many
SELECT d.id
FROM deals d
JOIN merchants m ON m.id = d.merchant_id AND m.status = 'active' AND m.deleted_at IS NULL
WHERE d.subcategory_id = sqlc.arg(subcategory_id)
  AND d.id <> sqlc.arg(exclude_deal_id)
  AND d.deleted_at IS NULL
  AND d.status IN ('active', 'sold_out')
  AND d.ends_at > now()
  AND (sqlc.arg(city)::text = '' OR EXISTS (
    SELECT 1
    FROM deal_locations dl
    JOIN merchant_locations ml ON ml.id = dl.merchant_location_id
    JOIN cities ci ON ci.id = ml.city_id
    WHERE dl.deal_id = d.id AND ci.slug = sqlc.arg(city)::text
  ))
ORDER BY d.sold_count DESC, d.id
LIMIT sqlc.arg(page_limit);

-- name: ListPublishedReviewsForDeal :many
SELECT r.id, r.deal_id, r.merchant_id, r.user_id, r.voucher_id, r.rating, r.text, r.locale, r.status,
       r.created_at, rr.id AS reply_id, rr.text AS reply_text, rr.created_at AS reply_created_at
FROM reviews r
LEFT JOIN review_replies rr ON rr.review_id = r.id
WHERE r.deal_id = sqlc.arg(deal_id) AND r.status = 'published' AND r.deleted_at IS NULL
ORDER BY r.created_at DESC, r.id DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountPublishedReviewsForDeal :one
SELECT COUNT(*) FROM reviews r
WHERE r.deal_id = $1 AND r.status = 'published' AND r.deleted_at IS NULL;

-- name: ListPublishedReviewsForMerchant :many
SELECT r.id, r.deal_id, r.merchant_id, r.user_id, r.voucher_id, r.rating, r.text, r.locale, r.status,
       r.created_at, rr.id AS reply_id, rr.text AS reply_text, rr.created_at AS reply_created_at
FROM reviews r
LEFT JOIN review_replies rr ON rr.review_id = r.id
WHERE r.merchant_id = $1 AND r.status = 'published' AND r.deleted_at IS NULL
ORDER BY r.created_at DESC, r.id DESC
LIMIT $2;

-- name: SuggestDeals :many
SELECT d.slug, d.title
FROM deals d
JOIN merchants m ON m.id = d.merchant_id AND m.status = 'active' AND m.deleted_at IS NULL
WHERE d.deleted_at IS NULL
  AND d.status IN ('active', 'sold_out')
  AND d.ends_at > now()
  AND (
    strpos(lower(immutable_unaccent(concat_ws(' ', d.title->>'uz', d.title->>'ru', d.title->>'en'))),
           lower(immutable_unaccent(sqlc.arg(query)::text))) > 0
    OR word_similarity(immutable_unaccent(sqlc.arg(query)::text),
         immutable_unaccent(concat_ws(' ', d.title->>'uz', d.title->>'ru', d.title->>'en'))) >= 0.35
  )
ORDER BY d.sold_count DESC, d.id
LIMIT sqlc.arg(page_limit);

-- name: SuggestMerchants :many
SELECT m.slug, m.display_name
FROM merchants m
WHERE m.status = 'active' AND m.deleted_at IS NULL
  AND (
    strpos(lower(immutable_unaccent(m.display_name)), lower(immutable_unaccent(sqlc.arg(query)::text))) > 0
    OR word_similarity(immutable_unaccent(sqlc.arg(query)::text), immutable_unaccent(m.display_name)) >= 0.35
  )
ORDER BY m.rating_count DESC, m.id
LIMIT sqlc.arg(page_limit);

-- name: SuggestCategories :many
SELECT slug, name
FROM categories
WHERE is_active = true
  AND (
    strpos(lower(immutable_unaccent(concat_ws(' ', name->>'uz', name->>'ru', name->>'en'))),
           lower(immutable_unaccent(sqlc.arg(query)::text))) > 0
    OR word_similarity(immutable_unaccent(sqlc.arg(query)::text),
         immutable_unaccent(concat_ws(' ', name->>'uz', name->>'ru', name->>'en'))) >= 0.35
  )
ORDER BY sort_order, slug
LIMIT sqlc.arg(page_limit);

-- name: ListActiveHomepageSections :many
SELECT id, key, title, type, deal_ids, rule, sort_order
FROM homepage_sections
WHERE is_active = true
  AND (starts_at IS NULL OR starts_at <= now())
  AND (ends_at IS NULL OR ends_at > now())
ORDER BY sort_order, key;
