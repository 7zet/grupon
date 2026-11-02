-- name: ListPendingDealIDs :many
SELECT id FROM deals
WHERE status = 'in_review' AND deleted_at IS NULL
ORDER BY updated_at ASC, id ASC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountPendingDeals :one
SELECT count(*) FROM deals WHERE status = 'in_review' AND deleted_at IS NULL;

-- name: GetDealReviewRow :one
SELECT id, merchant_id, status, starts_at FROM deals
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: TransitionPendingDeal :execrows
UPDATE deals SET status = sqlc.arg(status)::deal_status,
  rejection_reason = sqlc.narg(rejection_reason), updated_at = now()
WHERE id = sqlc.arg(id) AND status IN ('in_review', 'approved') AND deleted_at IS NULL;
