-- name: ReserveOptionStock :one
UPDATE deal_options
SET stock_reserved = stock_reserved + sqlc.arg(quantity), updated_at = now()
WHERE deal_options.id = sqlc.arg(option_id)
  AND is_active = true
  AND EXISTS (SELECT 1 FROM deals d JOIN categories c ON c.id = d.category_id WHERE d.id = deal_options.deal_id AND c.purchasable)
  AND stock_total - stock_sold - stock_reserved >= sqlc.arg(quantity)
RETURNING deal_options.id;

-- name: ReleaseOptionStock :exec
UPDATE deal_options
SET stock_reserved = stock_reserved - sqlc.arg(quantity), updated_at = now()
WHERE id = sqlc.arg(option_id) AND stock_reserved >= sqlc.arg(quantity);

-- name: ConvertReservedStock :exec
UPDATE deal_options
SET stock_reserved = stock_reserved - sqlc.arg(quantity),
    stock_sold = stock_sold + sqlc.arg(quantity),
    updated_at = now()
WHERE id = sqlc.arg(option_id) AND stock_reserved >= sqlc.arg(quantity);

-- name: LockUserForUpdate :one
SELECT id FROM users WHERE id = sqlc.arg(user_id) AND deleted_at IS NULL FOR UPDATE;

-- name: GetDealOptionRows :many
SELECT o.id AS option_id, o.deal_id, o.title AS option_title, o.original_price, o.deal_price,
       o.redemptions_allowed, o.stock_total, o.stock_sold, o.stock_reserved, o.is_active AS option_is_active,
       d.status AS deal_status, d.terms, d.starts_at, d.ends_at, d.max_per_user,
       d.voucher_valid_days, d.voucher_valid_until, d.merchant_id,
       (m.review_status = 'approved') AS merchant_active, c.purchasable AS category_purchasable
FROM deal_options o
JOIN deals d ON d.id = o.deal_id AND d.deleted_at IS NULL
JOIN merchants m ON m.id = d.merchant_id AND m.deleted_at IS NULL
JOIN categories c ON c.id = d.category_id
WHERE o.id = ANY(sqlc.arg(option_ids)::uuid[]);

-- name: UserUnitsForDeal :one
SELECT COALESCE(SUM(oi.quantity), 0)::int AS units
FROM order_items oi
JOIN orders o ON o.id = oi.order_id
WHERE o.user_id = sqlc.arg(user_id) AND oi.deal_id = sqlc.arg(deal_id)
  AND (o.status = 'paid' OR (o.status = 'awaiting_payment' AND EXISTS (
        SELECT 1 FROM stock_reservations r
        WHERE r.order_id = o.id AND r.released_at IS NULL AND r.expires_at > now())));

-- name: UpsertRecentlyViewed :exec
INSERT INTO recently_viewed (user_id, deal_id, viewed_at) VALUES (sqlc.arg(user_id), sqlc.arg(deal_id), now())
ON CONFLICT (user_id, deal_id) DO UPDATE SET viewed_at = now(), updated_at = now();

-- name: TrimRecentlyViewed :exec
DELETE FROM recently_viewed AS stale
WHERE stale.user_id = sqlc.arg(user_id)
  AND stale.id NOT IN (SELECT keep.id FROM recently_viewed AS keep
                       WHERE keep.user_id = sqlc.arg(user_id)
                       ORDER BY keep.viewed_at DESC, keep.id DESC LIMIT 30);

-- name: ListRecentlyViewedDealIDs :many
SELECT deal_id FROM recently_viewed
WHERE user_id = sqlc.arg(user_id) AND viewed_at > now() - interval '30 days'
ORDER BY viewed_at DESC, id DESC LIMIT 30;

-- name: NextOrderNumberSequence :one
SELECT nextval('order_number_seq')::bigint AS value;

-- name: InsertOrder :one
INSERT INTO orders (number, user_id, status, subtotal, total, currency, idempotency_key, request_hash)
VALUES (sqlc.arg(number), sqlc.arg(user_id), 'awaiting_payment', sqlc.arg(subtotal), sqlc.arg(total), 'UZS',
        sqlc.arg(idempotency_key), sqlc.arg(request_hash))
RETURNING id, created_at;

-- name: InsertOrderItem :one
INSERT INTO order_items (order_id, deal_id, deal_option_id, title_snapshot, unit_price_snapshot,
                         original_price_snapshot, terms_snapshot, quantity)
VALUES (sqlc.arg(order_id), sqlc.arg(deal_id), sqlc.arg(option_id), sqlc.arg(title_snapshot),
        sqlc.arg(unit_price), sqlc.arg(original_price), sqlc.arg(terms_snapshot), sqlc.arg(quantity))
RETURNING id;

-- name: InsertStockReservation :exec
INSERT INTO stock_reservations (deal_option_id, order_id, quantity, expires_at)
VALUES (sqlc.arg(option_id), sqlc.arg(order_id), sqlc.arg(quantity), sqlc.arg(expires_at));

-- name: GetOrderIDByIdempotencyKey :one
SELECT id FROM orders WHERE idempotency_key = sqlc.arg(idempotency_key);

-- name: GetOrderByID :one
SELECT o.id, o.number, o.user_id, o.status::text AS status, o.subtotal, o.total, o.paid_at,
       o.created_at, o.request_hash,
       (SELECT MAX(r.expires_at) FROM stock_reservations r
        WHERE r.order_id = o.id AND r.released_at IS NULL)::timestamptz AS reservation_expires_at
FROM orders o WHERE o.id = sqlc.arg(order_id);

-- name: LockOrderByID :one
SELECT id, number, user_id, status::text AS status, subtotal, total, paid_at, created_at
FROM orders WHERE id = sqlc.arg(order_id) FOR UPDATE;

-- name: ListOrderIDsForUser :many
SELECT id FROM orders WHERE user_id = sqlc.arg(user_id)
ORDER BY created_at DESC, id DESC LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CountOrdersForUser :one
SELECT COUNT(*)::bigint AS total FROM orders WHERE user_id = sqlc.arg(user_id);

-- name: ListOrderItemsForOrders :many
SELECT id, order_id, deal_id, deal_option_id, title_snapshot, unit_price_snapshot,
       original_price_snapshot, terms_snapshot, quantity
FROM order_items WHERE order_id = ANY(sqlc.arg(order_ids)::uuid[]) ORDER BY created_at, id;

-- name: SetOrderStatus :exec
UPDATE orders SET status = sqlc.arg(status)::order_status,
                  paid_at = COALESCE(sqlc.narg(paid_at), paid_at), updated_at = now()
WHERE id = sqlc.arg(order_id);

-- name: ListUnreleasedReservations :many
SELECT deal_option_id, quantity, expires_at FROM stock_reservations
WHERE order_id = sqlc.arg(order_id) AND released_at IS NULL ORDER BY deal_option_id;

-- name: ExtendReservations :exec
UPDATE stock_reservations SET expires_at = sqlc.arg(expires_at), updated_at = now()
WHERE order_id = sqlc.arg(order_id) AND released_at IS NULL;

-- name: MarkReservationsReleased :many
UPDATE stock_reservations SET released_at = now(), updated_at = now()
WHERE order_id = sqlc.arg(order_id) AND released_at IS NULL
RETURNING deal_option_id, quantity;

-- name: ExpiredUnpaidOrderIDs :many
SELECT DISTINCT o.id FROM orders o
JOIN stock_reservations r ON r.order_id = o.id
WHERE o.status = 'awaiting_payment' AND r.released_at IS NULL AND r.expires_at <= sqlc.arg(now)
ORDER BY o.id LIMIT sqlc.arg(batch_size);

-- name: InsertPayment :exec
INSERT INTO payments (id, order_id, provider, provider_payment_id, status, amount, currency, raw_payload)
VALUES (sqlc.arg(id), sqlc.arg(order_id), sqlc.arg(provider)::payment_provider, sqlc.arg(provider_payment_id),
        sqlc.arg(status)::payment_status, sqlc.arg(amount), 'UZS', sqlc.arg(raw_payload));

-- name: GetPaymentByID :one
SELECT id, order_id, provider::text AS provider, provider_payment_id, status::text AS status, amount, raw_payload
FROM payments WHERE id = sqlc.arg(id);

-- name: LockPaymentByProviderID :one
SELECT id, order_id, provider::text AS provider, provider_payment_id, status::text AS status, amount, raw_payload
FROM payments WHERE provider = sqlc.arg(provider)::payment_provider AND provider_payment_id = sqlc.arg(provider_payment_id)
FOR UPDATE;

-- name: SetPaymentStatus :exec
UPDATE payments SET status = sqlc.arg(status)::payment_status,
                    raw_payload = COALESCE(sqlc.narg(raw_payload)::jsonb, raw_payload), updated_at = now()
WHERE id = sqlc.arg(id);

-- name: RecordManualPaymentConfirmation :exec
INSERT INTO audit_logs(actor_user_id, actor_role, action, entity_type, entity_id, after_data)
VALUES (sqlc.arg(actor_id), sqlc.arg(actor_role)::app_role, 'manual_payment_confirmed', 'payment',
        sqlc.arg(payment_id), jsonb_build_object('reference', sqlc.arg(reference)::text, 'note', sqlc.arg(note)::text));

-- name: InsertVoucher :exec
INSERT INTO vouchers (id, order_item_id, seq, user_id, deal_id, deal_option_id, merchant_id, code, qr_token,
                      status, valid_from, valid_until, redemptions_allowed, zaxira_code_hmac)
VALUES (sqlc.arg(id), sqlc.arg(order_item_id), sqlc.arg(seq), sqlc.arg(user_id), sqlc.arg(deal_id), sqlc.arg(option_id),
        sqlc.arg(merchant_id), sqlc.arg(code), sqlc.arg(qr_token), 'active', sqlc.arg(valid_from), sqlc.arg(valid_until),
        sqlc.arg(redemptions_allowed), sqlc.arg(zaxira_code_hmac));

-- name: VoucherCodeExists :one
SELECT EXISTS (SELECT 1 FROM vouchers WHERE code = sqlc.arg(code)) AS taken;

-- name: ListVouchersForUser :many
SELECT v.id, v.order_item_id, v.user_id, v.deal_id, v.deal_option_id, v.merchant_id, v.code, v.qr_token,
       v.status::text AS status, v.valid_from, v.valid_until, v.redeemed_at, v.redeemed_location_id,
       oi.terms_snapshot
FROM vouchers v JOIN order_items oi ON oi.id = v.order_item_id
WHERE v.user_id = sqlc.arg(user_id)
  AND (sqlc.narg(status)::text IS NULL OR v.status::text = sqlc.narg(status)::text)
ORDER BY v.valid_until, v.id
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CountVouchersForUser :one
SELECT COUNT(*)::bigint AS total FROM vouchers
WHERE user_id = sqlc.arg(user_id)
  AND (sqlc.narg(status)::text IS NULL OR status::text = sqlc.narg(status)::text);

-- name: GetVoucherForUser :one
SELECT v.id, v.order_item_id, v.user_id, v.deal_id, v.deal_option_id, v.merchant_id, v.code, v.qr_token,
       v.status::text AS status, v.valid_from, v.valid_until, v.redeemed_at, v.redeemed_location_id,
       oi.terms_snapshot
FROM vouchers v JOIN order_items oi ON oi.id = v.order_item_id
WHERE v.id = sqlc.arg(id) AND v.user_id = sqlc.arg(user_id);

-- name: ExpireDueVouchers :many
UPDATE vouchers SET status = 'expired', updated_at = now()
WHERE id IN (SELECT v.id FROM vouchers v WHERE v.status = 'active' AND v.valid_until <= sqlc.arg(now)
             ORDER BY v.valid_until LIMIT sqlc.arg(batch_size) FOR UPDATE SKIP LOCKED)
RETURNING id;

-- name: AddDealSoldCount :exec
UPDATE deals SET sold_count = sold_count + sqlc.arg(quantity), updated_at = now()
WHERE id = sqlc.arg(deal_id);

-- name: GetUserContact :one
SELECT email, name, locale::text AS locale FROM users WHERE id = sqlc.arg(user_id) AND deleted_at IS NULL;

-- name: FailOpenPayments :execrows
UPDATE payments SET status = 'failed', updated_at = now()
WHERE order_id = sqlc.arg(order_id) AND status IN ('created', 'pending');

-- name: ListBuyerNotifications :many
SELECT id, kind, voucher_id, merchant_name, created_at, read_at
FROM notifications WHERE user_id = sqlc.arg(user_id)
ORDER BY created_at DESC LIMIT 50;
