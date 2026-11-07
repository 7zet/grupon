-- name: LockVoucherForRedemption :one
SELECT v.id, v.user_id, v.status, v.redemptions_used, v.redemptions_allowed, v.valid_from, v.valid_until,
       v.fraud_hold, v.dispute_open,
       oi.unit_price_snapshot, m.commission_bps, m.display_name AS merchant_name, c.redeemable AS category_redeemable
FROM vouchers v
JOIN order_items oi ON oi.id = v.order_item_id
JOIN merchants m ON m.id = v.merchant_id
JOIN deals d ON d.id = v.deal_id
JOIN categories c ON c.id = d.category_id
WHERE v.id = sqlc.arg(id) AND v.merchant_id = sqlc.arg(merchant_id)
FOR UPDATE OF v;

-- name: ApplyVoucherUse :execrows
UPDATE vouchers SET
  redemptions_used = redemptions_used + 1,
  status = CASE WHEN redemptions_used + 1 >= redemptions_allowed THEN 'redeemed'::voucher_status WHEN redemptions_allowed > 1 THEN 'partially_redeemed'::voucher_status ELSE status END,
  redeemed_at = CASE WHEN redemptions_used + 1 >= redemptions_allowed THEN now() ELSE redeemed_at END,
  redeemed_by_user_id = sqlc.arg(staff_user_id),
  redeemed_location_id = sqlc.arg(location_id),
  updated_at = now()
WHERE id = sqlc.arg(id) AND status IN ('active', 'partially_redeemed') AND redemptions_used < redemptions_allowed;

-- name: InsertRedemption :one
INSERT INTO redemptions (voucher_id, merchant_id, merchant_location_id, staff_user_id, sequence, amount, commission, geo_lat, geo_lng)
VALUES (sqlc.arg(voucher_id), sqlc.arg(merchant_id), sqlc.arg(location_id), sqlc.arg(staff_user_id),
        sqlc.arg(sequence), sqlc.arg(amount), sqlc.arg(commission), sqlc.narg(geo_lat)::float8, sqlc.narg(geo_lng)::float8)
RETURNING id;

-- name: LatestOpenRedemption :one
SELECT r.id, r.voucher_id, r.merchant_id, r.staff_user_id, r.sequence, r.amount, r.commission, r.created_at,
       CASE WHEN pr.id IS NOT NULL AND pr.status::text IN ('sent', 'reconciled') THEN true ELSE false END AS payout_sent
FROM redemptions r LEFT JOIN payout_runs pr ON pr.id = r.payout_run_id
WHERE r.voucher_id = sqlc.arg(voucher_id) AND r.merchant_id = sqlc.arg(merchant_id) AND r.reversed_at IS NULL
ORDER BY sequence DESC
LIMIT 1
FOR UPDATE OF r;

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

-- name: VoucherIDByBackupCodeHMAC :one
SELECT id FROM vouchers WHERE zaxira_code_hmac = sqlc.arg(zaxira_code_hmac);

-- name: MerchantLocationCoords :one
SELECT lat::float8 AS lat, lng::float8 AS lng FROM merchant_locations WHERE id = sqlc.arg(id) AND merchant_id = sqlc.arg(merchant_id);

-- name: LocationAttachedToDeal :one
SELECT EXISTS (
  SELECT 1 FROM deal_locations WHERE deal_id = sqlc.arg(deal_id) AND merchant_location_id = sqlc.arg(location_id)
);

-- name: LatestOpenRedemptionForVoucher :one
SELECT r.id, r.voucher_id, r.merchant_id, r.staff_user_id, r.sequence, r.amount, r.commission, r.created_at,
       CASE WHEN pr.id IS NOT NULL AND pr.status::text IN ('sent', 'reconciled') THEN true ELSE false END AS payout_sent
FROM redemptions r LEFT JOIN payout_runs pr ON pr.id = r.payout_run_id
WHERE r.voucher_id = sqlc.arg(voucher_id) AND r.reversed_at IS NULL
ORDER BY sequence DESC
LIMIT 1
FOR UPDATE OF r;

-- name: InsertBuyerNotification :exec
INSERT INTO notifications (user_id, kind, voucher_id, merchant_name)
VALUES (sqlc.arg(user_id), 'voucher_redeemed', sqlc.arg(voucher_id), sqlc.arg(merchant_name));

-- name: NextRedemptionSequence :one
SELECT (COALESCE(MAX(sequence), 0) + 1)::int AS next_sequence FROM redemptions WHERE voucher_id = sqlc.arg(voucher_id);

-- name: RedemptionAttemptStats :one
SELECT COUNT(*) FILTER (WHERE created_at > sqlc.arg(minute_since))::int AS last_minute,
       COUNT(*) FILTER (WHERE failed)::int AS failed_recent
FROM redemption_attempts
WHERE staff_user_id = sqlc.arg(staff_user_id) AND created_at > sqlc.arg(block_since);

-- name: InsertRedemptionAttempt :exec
INSERT INTO redemption_attempts (staff_user_id, failed) VALUES (sqlc.arg(staff_user_id), sqlc.arg(failed));

-- name: CompleteFullyUsedOrder :execrows
-- PRD 08: a paid order becomes completed once none of its vouchers can still be redeemed.
UPDATE orders o SET status = 'completed', updated_at = now()
WHERE o.id = (SELECT oi.order_id FROM vouchers v JOIN order_items oi ON oi.id = v.order_item_id WHERE v.id = sqlc.arg(voucher_id))
  AND o.status = 'paid'
  AND NOT EXISTS (
    SELECT 1 FROM vouchers v JOIN order_items oi ON oi.id = v.order_item_id
    WHERE oi.order_id = o.id AND v.status IN ('issued', 'active', 'partially_redeemed')
  );

-- name: HoldBurstVouchers :execrows
-- PRD 12.1: vouchers scanned by a staff member during a redemption burst wait for fraud review.
UPDATE vouchers SET fraud_hold = true, updated_at = now()
WHERE id IN (
  SELECT r.voucher_id FROM redemptions r
  WHERE r.staff_user_id = sqlc.arg(staff_user_id) AND r.created_at >= sqlc.arg(since) AND r.reversed_at IS NULL
);
