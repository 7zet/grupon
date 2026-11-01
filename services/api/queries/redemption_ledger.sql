-- name: LockVoucherForRedemption :one
SELECT v.id, v.status, v.redemptions_used, v.redemptions_allowed, v.valid_from, v.valid_until,
       oi.unit_price_snapshot, m.commission_bps
FROM vouchers v
JOIN order_items oi ON oi.id = v.order_item_id
JOIN merchants m ON m.id = v.merchant_id
WHERE v.id = sqlc.arg(id) AND v.merchant_id = sqlc.arg(merchant_id)
FOR UPDATE OF v;

-- name: ApplyVoucherUse :execrows
UPDATE vouchers SET
  redemptions_used = redemptions_used + 1,
  status = CASE WHEN redemptions_used + 1 >= redemptions_allowed THEN 'redeemed'::voucher_status ELSE status END,
  redeemed_at = CASE WHEN redemptions_used + 1 >= redemptions_allowed THEN now() ELSE redeemed_at END,
  redeemed_by_user_id = sqlc.arg(staff_user_id),
  redeemed_location_id = sqlc.arg(location_id),
  updated_at = now()
WHERE id = sqlc.arg(id) AND status = 'active';

-- name: InsertRedemption :one
INSERT INTO redemptions (voucher_id, merchant_id, merchant_location_id, staff_user_id, sequence, amount, commission)
VALUES (sqlc.arg(voucher_id), sqlc.arg(merchant_id), sqlc.arg(location_id), sqlc.arg(staff_user_id),
        sqlc.arg(sequence), sqlc.arg(amount), sqlc.arg(commission))
RETURNING id;

-- name: LatestOpenRedemption :one
SELECT id, voucher_id, merchant_id, staff_user_id, sequence, amount, commission, created_at
FROM redemptions
WHERE voucher_id = sqlc.arg(voucher_id) AND merchant_id = sqlc.arg(merchant_id) AND reversed_at IS NULL
ORDER BY sequence DESC
LIMIT 1
FOR UPDATE;

-- name: MarkRedemptionReversed :execrows
UPDATE redemptions SET reversed_at = now(), reversed_by_user_id = sqlc.arg(actor_user_id), reversal_reason = sqlc.arg(reason)
WHERE id = sqlc.arg(id) AND reversed_at IS NULL;

-- name: RollBackVoucherUse :execrows
UPDATE vouchers SET
  redemptions_used = redemptions_used - 1,
  status = 'active',
  redeemed_at = NULL,
  updated_at = now()
WHERE id = sqlc.arg(id) AND redemptions_used > 0;
