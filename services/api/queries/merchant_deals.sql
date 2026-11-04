-- name: InsertDeal :one
INSERT INTO deals (slug, merchant_id, category_id, subcategory_id, title, short_description, description,
  highlights, whats_included, terms, faq, redemption_method, redemption_instructions, booking_required,
  status, starts_at, ends_at, voucher_valid_days, voucher_valid_until, max_per_user, review_fingerprint)
VALUES (sqlc.arg(slug), sqlc.arg(merchant_id), sqlc.arg(category_id), sqlc.arg(subcategory_id),
  sqlc.arg(title), sqlc.arg(short_description), sqlc.arg(description),
  sqlc.arg(highlights), sqlc.arg(whats_included), sqlc.arg(terms), sqlc.arg(faq),
  sqlc.arg(redemption_method)::redemption_method, sqlc.arg(redemption_instructions),
  sqlc.arg(booking_required), sqlc.arg(status)::deal_status,
  sqlc.arg(starts_at), sqlc.arg(ends_at), sqlc.narg(voucher_valid_days), sqlc.narg(voucher_valid_until),
  sqlc.arg(max_per_user), sqlc.arg(review_fingerprint))
RETURNING id;

-- name: UpdateOwnedDeal :execrows
UPDATE deals SET slug = sqlc.arg(slug), category_id = sqlc.arg(category_id), subcategory_id = sqlc.arg(subcategory_id),
  title = sqlc.arg(title), short_description = sqlc.arg(short_description),
  description = sqlc.arg(description), highlights = sqlc.arg(highlights), whats_included = sqlc.arg(whats_included),
  terms = sqlc.arg(terms), faq = sqlc.arg(faq), redemption_method = sqlc.arg(redemption_method)::redemption_method,
  redemption_instructions = sqlc.arg(redemption_instructions), booking_required = sqlc.arg(booking_required),
  status = sqlc.arg(status)::deal_status, rejection_reason = NULL,
  starts_at = sqlc.arg(starts_at), ends_at = sqlc.arg(ends_at), voucher_valid_days = sqlc.narg(voucher_valid_days),
  voucher_valid_until = sqlc.narg(voucher_valid_until), max_per_user = sqlc.arg(max_per_user),
  review_fingerprint = sqlc.arg(review_fingerprint), updated_at = now()
WHERE id = sqlc.arg(id) AND merchant_id = sqlc.arg(merchant_id) AND deleted_at IS NULL;

-- name: SetOwnedDealStatus :execrows
UPDATE deals SET status = sqlc.arg(status)::deal_status, updated_at = now()
WHERE id = sqlc.arg(id) AND merchant_id = sqlc.arg(merchant_id) AND deleted_at IS NULL;

-- name: SoftDeleteOwnedDraftDeal :execrows
UPDATE deals SET deleted_at = now(), updated_at = now()
WHERE id = sqlc.arg(id) AND merchant_id = sqlc.arg(merchant_id) AND status = 'draft' AND deleted_at IS NULL;

-- name: GetOwnedDealRecord :one
SELECT id, status, review_fingerprint, category_id
FROM deals
WHERE id = sqlc.arg(id) AND merchant_id = sqlc.arg(merchant_id) AND deleted_at IS NULL;

-- name: GetOwnedDealDetailRow :one
SELECT d.id, d.slug, d.title, d.status, d.sold_count, d.rating_avg::float8 AS rating_avg,
       d.rating_count, d.starts_at, d.ends_at, d.short_description, d.description, d.highlights,
       d.whats_included, d.terms, d.faq, d.redemption_method, d.redemption_instructions,
       d.booking_required, d.voucher_valid_days, d.voucher_valid_until, d.max_per_user,
       d.merchant_id, d.category_id, d.subcategory_id, sc.name AS subcategory_name, sc.slug AS subcategory_slug
FROM deals d
JOIN subcategories sc ON sc.id = d.subcategory_id
WHERE d.id = sqlc.arg(id) AND d.merchant_id = sqlc.arg(merchant_id) AND d.deleted_at IS NULL;

-- name: UpsertDealOption :exec
INSERT INTO deal_options (id, deal_id, title, original_price, deal_price, stock_total, sort_order, is_active, redemptions_allowed)
VALUES (sqlc.arg(id), sqlc.arg(deal_id), sqlc.arg(title), sqlc.arg(original_price), sqlc.arg(deal_price),
        sqlc.arg(stock_total), sqlc.arg(sort_order), sqlc.arg(is_active), sqlc.arg(redemptions_allowed))
ON CONFLICT (id) DO UPDATE SET title = EXCLUDED.title, original_price = EXCLUDED.original_price,
  deal_price = EXCLUDED.deal_price, stock_total = EXCLUDED.stock_total, sort_order = EXCLUDED.sort_order,
  redemptions_allowed = EXCLUDED.redemptions_allowed,
  is_active = EXCLUDED.is_active, updated_at = now()
WHERE deal_options.deal_id = EXCLUDED.deal_id;

-- name: DeactivateDealOptionsExcept :exec
UPDATE deal_options SET is_active = false, updated_at = now()
WHERE deal_id = sqlc.arg(deal_id) AND NOT (id = ANY(sqlc.arg(keep_ids)::uuid[]));

-- name: DeleteDealMedia :exec
DELETE FROM deal_media WHERE deal_id = sqlc.arg(deal_id);

-- name: InsertDealMedia :exec
INSERT INTO deal_media (deal_id, url, alt, sort_order, width, height, blurhash)
VALUES (sqlc.arg(deal_id), sqlc.arg(url), sqlc.arg(alt), sqlc.arg(sort_order), sqlc.arg(width), sqlc.arg(height), sqlc.arg(blurhash));

-- name: DeleteDealLocations :exec
DELETE FROM deal_locations WHERE deal_id = sqlc.arg(deal_id);

-- name: InsertDealLocation :exec
INSERT INTO deal_locations (deal_id, merchant_location_id) VALUES (sqlc.arg(deal_id), sqlc.arg(merchant_location_id));

-- name: CountOwnedLocations :one
SELECT count(*) FROM merchant_locations
WHERE merchant_id = sqlc.arg(merchant_id) AND id = ANY(sqlc.arg(location_ids)::uuid[]);

-- name: SubcategoryCategoryID :one
SELECT category_id FROM subcategories WHERE id = sqlc.arg(id);

-- name: CountOwnedDeals :one
SELECT count(*) FROM deals
WHERE merchant_id = sqlc.arg(merchant_id) AND deleted_at IS NULL
  AND (sqlc.narg(status)::deal_status IS NULL OR status = sqlc.narg(status)::deal_status);

-- name: ListOwnedDealIDs :many
SELECT id FROM deals
WHERE merchant_id = sqlc.arg(merchant_id) AND deleted_at IS NULL
  AND (sqlc.narg(status)::deal_status IS NULL OR status = sqlc.narg(status)::deal_status)
ORDER BY updated_at DESC, id DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: InsertAuditLog :exec
INSERT INTO audit_logs (actor_user_id, actor_role, action, entity_type, entity_id, after_data)
VALUES (sqlc.arg(actor_user_id), sqlc.arg(actor_role)::app_role, sqlc.arg(action), sqlc.arg(entity_type),
        sqlc.arg(entity_id), sqlc.arg(after_data));
