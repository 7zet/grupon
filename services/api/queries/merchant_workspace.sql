-- Merchant workspace: the signed-in merchant's business, branches and dashboard aggregates.
-- Every query is scoped by merchant_id; callers never pass an unscoped id.

-- name: MerchantRoleForUser :one
SELECT ra.merchant_id, ra.role::text AS role
FROM role_assignments ra
WHERE ra.user_id = $1 AND ra.merchant_id IS NOT NULL
ORDER BY ra.created_at, ra.id
LIMIT 1;

-- name: GetMerchantWorkspace :one
SELECT id, slug, legal_name, display_name, description, logo_url, cover_url, phone, website,
       instagram, telegram, status, review_status, rating_avg::float8 AS rating_avg, rating_count, commission_bps
FROM merchants
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateMerchantWorkspace :one
UPDATE merchants
SET display_name = $2, description = $3, logo_url = $4, cover_url = $5, phone = $6,
    website = $7, instagram = $8, telegram = $9, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, slug, legal_name, display_name, description, logo_url, cover_url, phone, website,
          instagram, telegram, status, rating_avg::float8 AS rating_avg, rating_count, commission_bps;

-- name: CreateMerchantLocationWithHours :one
INSERT INTO merchant_locations (merchant_id, city_id, address, lat, lng, opening_hours, phone)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id;

-- name: UpdateMerchantLocationRow :execrows
UPDATE merchant_locations
SET city_id = $3, address = $4, lat = $5, lng = $6, opening_hours = $7, phone = $8, updated_at = now()
WHERE merchant_locations.id = $1 AND merchant_locations.merchant_id = $2;

-- name: DeleteMerchantLocationRow :execrows
DELETE FROM merchant_locations
WHERE merchant_locations.id = $1 AND merchant_locations.merchant_id = $2
  AND NOT EXISTS (
    SELECT 1 FROM deal_locations dl JOIN deals d ON d.id = dl.deal_id
    WHERE dl.merchant_location_id = merchant_locations.id AND d.status IN ('scheduled', 'active', 'sold_out')
  );

-- name: MerchantLocationBelongsTo :one
SELECT EXISTS (SELECT 1 FROM merchant_locations WHERE id = $1 AND merchant_id = $2);

-- name: DashboardSales :one
SELECT COALESCE(SUM(oi.unit_price_snapshot * oi.quantity), 0)::bigint AS total
FROM order_items oi
JOIN orders o ON o.id = oi.order_id
JOIN deals d ON d.id = oi.deal_id
WHERE d.merchant_id = $1 AND o.status = 'paid' AND o.paid_at >= $2 AND o.paid_at < $3;

-- name: DashboardVouchersSold :one
SELECT COUNT(*)::bigint FROM vouchers
WHERE merchant_id = $1 AND created_at >= $2 AND created_at < $3;

-- name: DashboardVouchersRedeemed :one
SELECT COUNT(*)::bigint FROM vouchers
WHERE merchant_id = $1 AND redeemed_at >= $2 AND redeemed_at < $3;

-- name: DashboardActiveDeals :one
SELECT COUNT(*)::bigint FROM deals
WHERE merchant_id = $1 AND status = 'active' AND deleted_at IS NULL;

-- name: DashboardSalesChart :many
SELECT (o.paid_at AT TIME ZONE 'Asia/Tashkent')::date AS day,
       SUM(oi.unit_price_snapshot * oi.quantity)::bigint AS total
FROM order_items oi
JOIN orders o ON o.id = oi.order_id
JOIN deals d ON d.id = oi.deal_id
WHERE d.merchant_id = $1 AND o.status = 'paid' AND o.paid_at >= $2 AND o.paid_at < $3
GROUP BY day
ORDER BY day;

-- name: DashboardLatestOrderIDs :many
SELECT o.id
FROM orders o
JOIN order_items oi ON oi.order_id = o.id
JOIN deals d ON d.id = oi.deal_id
WHERE d.merchant_id = $1 AND o.status = 'paid'
GROUP BY o.id, o.paid_at
ORDER BY o.paid_at DESC
LIMIT $2;

-- name: DashboardPendingDealIDs :many
SELECT id FROM deals
WHERE merchant_id = $1 AND status = 'pending_review' AND deleted_at IS NULL
ORDER BY updated_at DESC
LIMIT $2;
