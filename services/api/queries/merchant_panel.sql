-- name: MerchantPayoutRows :many
SELECT pr.id,
       COALESCE(MIN(r.created_at), pr.created_at)::date AS period_start,
       pr.created_at::date AS period_end,
       COALESCE(SUM(r.amount), 0)::bigint AS gross,
       COALESCE(SUM(r.commission), 0)::bigint AS commission,
       pr.amount AS net,
       pr.status,
       pr.updated_at AS paid_at
FROM payout_runs pr
LEFT JOIN redemptions r ON r.payout_run_id = pr.id
WHERE pr.merchant_id = sqlc.arg(merchant_id)
GROUP BY pr.id
ORDER BY pr.created_at DESC
LIMIT 100;

-- name: MerchantCustomerRows :many
SELECT u.id AS user_id,
       u.name,
       COUNT(DISTINCT v.order_item_id)::int AS orders_count,
       COUNT(DISTINCT v.id)::int AS vouchers_count,
       COALESCE(SUM(oi.unit_price_snapshot), 0)::bigint AS total_spent,
       MAX(o.created_at) AS last_order_at
FROM vouchers v
JOIN users u ON u.id = v.user_id
JOIN order_items oi ON oi.id = v.order_item_id
JOIN orders o ON o.id = oi.order_id
WHERE v.merchant_id = sqlc.arg(merchant_id)
GROUP BY u.id, u.name
ORDER BY MAX(o.created_at) DESC
LIMIT 100;

-- name: MerchantSoldTotals :one
SELECT COUNT(*)::bigint AS vouchers_sold,
       COALESCE(SUM(oi.unit_price_snapshot), 0)::bigint AS gross_sales
FROM vouchers v
JOIN order_items oi ON oi.id = v.order_item_id
WHERE v.merchant_id = sqlc.arg(merchant_id)
  AND v.created_at >= sqlc.arg(from_at) AND v.created_at < sqlc.arg(to_at);

-- name: MerchantRedeemedCount :one
SELECT COUNT(*)::bigint AS redeemed
FROM redemptions
WHERE merchant_id = sqlc.arg(merchant_id) AND reversed_at IS NULL
  AND created_at >= sqlc.arg(from_at) AND created_at < sqlc.arg(to_at);

-- name: MerchantSalesByDay :many
SELECT (v.created_at AT TIME ZONE 'Asia/Tashkent')::date AS day,
       COUNT(*)::bigint AS vouchers,
       COALESCE(SUM(oi.unit_price_snapshot), 0)::bigint AS amount
FROM vouchers v
JOIN order_items oi ON oi.id = v.order_item_id
WHERE v.merchant_id = sqlc.arg(merchant_id)
  AND v.created_at >= sqlc.arg(from_at) AND v.created_at < sqlc.arg(to_at)
GROUP BY day
ORDER BY day;

-- name: MerchantReviewRows :many
SELECT r.id, r.deal_id, r.merchant_id, r.user_id, r.voucher_id, r.rating, r.text, r.locale, r.status,
       r.created_at, rr.id AS reply_id, rr.text AS reply_text, rr.created_at AS reply_created_at, rr.updated_at AS reply_updated_at
FROM reviews r
LEFT JOIN review_replies rr ON rr.review_id = r.id
WHERE r.merchant_id = sqlc.arg(merchant_id) AND r.deleted_at IS NULL
ORDER BY r.created_at DESC
LIMIT 100;

-- name: ReviewReplyByReview :one
SELECT id, text, created_at, updated_at FROM review_replies WHERE review_id = sqlc.arg(review_id);

-- name: InsertReviewReply :one
INSERT INTO review_replies (review_id, merchant_user_id, text)
VALUES (sqlc.arg(review_id), sqlc.arg(merchant_user_id), sqlc.arg(text))
RETURNING id, text, created_at, updated_at;

-- name: UpdateReviewReply :one
UPDATE review_replies SET text = sqlc.arg(text), updated_at = now()
WHERE review_id = sqlc.arg(review_id)
RETURNING id, text, created_at, updated_at;

-- name: MerchantOrderRows :many
SELECT DISTINCT o.id, o.number, o.status::text AS status, o.subtotal, o.total, o.created_at
FROM orders o
JOIN order_items oi ON oi.order_id = o.id
JOIN deals d ON d.id = oi.deal_id
WHERE d.merchant_id = sqlc.arg(merchant_id)
ORDER BY o.created_at DESC
LIMIT 100;

-- name: MerchantSettingsRow :one
SELECT order_emails, review_emails, payout_emails, default_location_id FROM merchants
WHERE id = sqlc.arg(merchant_id) AND deleted_at IS NULL;

-- name: UpdateMerchantSettingsRow :execrows
UPDATE merchants SET order_emails = sqlc.arg(order_emails), review_emails = sqlc.arg(review_emails),
  payout_emails = sqlc.arg(payout_emails), default_location_id = sqlc.narg(default_location_id), updated_at = now()
WHERE id = sqlc.arg(merchant_id) AND deleted_at IS NULL;
