-- name: ListMerchantRedemptionHistory :many
SELECT r.id, r.created_at, d.title AS deal_title, o.title AS option_title,
       buyer.name AS buyer_name, ml.address AS branch, cashier.name AS cashier_name,
       CASE WHEN r.reversed_at IS NOT NULL THEN true ELSE false END AS reversed
FROM redemptions r
JOIN vouchers v ON v.id = r.voucher_id
JOIN users buyer ON buyer.id = v.user_id
JOIN users cashier ON cashier.id = r.staff_user_id
JOIN deals d ON d.id = v.deal_id
JOIN deal_options o ON o.id = v.deal_option_id
JOIN merchant_locations ml ON ml.id = r.merchant_location_id
WHERE r.merchant_id = sqlc.arg(merchant_id)
  AND (sqlc.narg(staff_user_id)::uuid IS NULL OR r.staff_user_id = sqlc.narg(staff_user_id)::uuid)
ORDER BY r.created_at DESC, r.id DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountMerchantRedemptionHistory :one
SELECT COUNT(*)::bigint FROM redemptions r
WHERE r.merchant_id = sqlc.arg(merchant_id)
  AND (sqlc.narg(staff_user_id)::uuid IS NULL OR r.staff_user_id = sqlc.narg(staff_user_id)::uuid);
