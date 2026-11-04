-- name: GetVoucherByID :one
SELECT id, order_item_id, user_id, deal_id, deal_option_id, merchant_id, code, qr_token, status,
       valid_from, valid_until, redeemed_at
FROM vouchers WHERE id = sqlc.arg(id);

-- name: GetVoucherByCode :one
SELECT id, order_item_id, user_id, deal_id, deal_option_id, merchant_id, code, qr_token, status,
       valid_from, valid_until, redeemed_at
FROM vouchers WHERE code = sqlc.arg(code);

-- name: RedeemVoucherIfActive :execrows
UPDATE vouchers SET status = 'redeemed', redeemed_at = sqlc.arg(redeemed_at),
  redeemed_by_user_id = sqlc.arg(redeemed_by_user_id), redeemed_location_id = sqlc.arg(location_id),
  updated_at = now()
WHERE id = sqlc.arg(id) AND merchant_id = sqlc.arg(merchant_id) AND status = 'active'
  AND valid_from <= sqlc.arg(redeemed_at) AND valid_until > sqlc.arg(redeemed_at);
