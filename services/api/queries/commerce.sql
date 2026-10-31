-- name: ReserveOptionStock :one
UPDATE deal_options
SET stock_reserved = stock_reserved + sqlc.arg(quantity)
WHERE id = sqlc.arg(option_id)
  AND is_active = true
  AND stock_total - stock_sold - stock_reserved >= sqlc.arg(quantity)
RETURNING *;

-- name: ReleaseOptionStock :one
UPDATE deal_options
SET stock_reserved = stock_reserved - sqlc.arg(quantity)
WHERE id = sqlc.arg(option_id) AND stock_reserved >= sqlc.arg(quantity)
RETURNING *;

-- name: ConvertReservedStock :one
UPDATE deal_options
SET stock_reserved = stock_reserved - sqlc.arg(quantity), stock_sold = stock_sold + sqlc.arg(quantity)
WHERE id = sqlc.arg(option_id) AND stock_reserved >= sqlc.arg(quantity)
RETURNING *;

-- name: GetOrderForUser :one
SELECT * FROM orders WHERE id = $1 AND user_id = $2;

-- name: ListVouchersForUser :many
SELECT * FROM vouchers WHERE user_id = $1 AND status = sqlc.arg(status) ORDER BY created_at DESC LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: GetVoucherByCode :one
SELECT * FROM vouchers WHERE code = $1;

-- name: GetVoucherByQRToken :one
SELECT * FROM vouchers WHERE qr_token = $1;
