-- name: CountOpenReviewReports :one
SELECT count(*) FROM review_reports WHERE status = 'pending';

-- name: ListOpenReviewReports :many
SELECT rr.id, rr.review_id, rr.reason, rr.status, rr.created_at
FROM review_reports rr
WHERE rr.status = 'pending'
ORDER BY rr.created_at ASC, rr.id ASC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: GetReviewReportRow :one
SELECT id, review_id, reason, status, created_at FROM review_reports WHERE id = sqlc.arg(id);

-- name: DismissOpenReviewReport :execrows
UPDATE review_reports SET status = 'dismissed', updated_at = now()
WHERE id = sqlc.arg(id) AND status = 'pending';

-- name: HideReviewRow :execrows
UPDATE reviews SET status = 'hidden', updated_at = now()
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: ActionOpenReviewReports :exec
UPDATE review_reports SET status = 'resolved', updated_at = now()
WHERE review_id = sqlc.arg(review_id) AND status = 'pending';

-- name: CountAuditLogs :one
SELECT count(*) FROM audit_logs;

-- name: ListAuditLogs :many
SELECT id, actor_user_id, actor_role, action, entity_type, entity_id, after_data, created_at
FROM audit_logs
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);
