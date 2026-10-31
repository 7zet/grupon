-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1 AND deleted_at IS NULL;

-- name: GetUserByGoogleSub :one
SELECT * FROM users WHERE google_sub = $1 AND deleted_at IS NULL;

-- name: ListUserRoles :many
SELECT * FROM role_assignments WHERE user_id = $1 ORDER BY role;

-- name: RevokeRefreshFamily :exec
UPDATE refresh_tokens SET revoked_at = now() WHERE family_id = $1 AND revoked_at IS NULL;

-- name: ListMerchantStaff :many
SELECT ra.* FROM role_assignments ra WHERE ra.merchant_id = $1 AND ra.role IN ('merchant_owner', 'merchant_manager', 'merchant_cashier') ORDER BY ra.created_at;
