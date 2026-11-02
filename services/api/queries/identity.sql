-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1 AND deleted_at IS NULL;

-- name: ListUserRoles :many
SELECT * FROM role_assignments WHERE user_id = $1 ORDER BY role;

-- name: RevokeRefreshFamily :exec
UPDATE refresh_tokens SET revoked_at = now() WHERE family_id = $1 AND revoked_at IS NULL;

-- name: ListMerchantStaff :many
SELECT ra.* FROM role_assignments ra WHERE ra.merchant_id = $1 AND ra.role IN ('merchant_owner', 'merchant_manager', 'merchant_cashier') ORDER BY ra.created_at;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1 AND deleted_at IS NULL;

-- name: CreateUser :one
INSERT INTO users (email, name, locale, currency, email_verified_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: MarkUserEmailVerified :exec
UPDATE users SET email_verified_at = COALESCE(email_verified_at, now()) WHERE id = $1;

-- name: UpdateUserProfile :one
UPDATE users SET name = $2, phone = $3, locale = $4, currency = $5
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: TouchUserLastLogin :exec
UPDATE users SET last_login_at = now() WHERE id = $1;

-- name: SoftDeleteUser :exec
-- Deleted accounts keep a tombstone email so the address can be reused by a new account.
UPDATE users SET deleted_at = now(), email = 'deleted+' || id::text || '@deleted.invalid'
WHERE id = $1 AND deleted_at IS NULL;

-- name: CreateRoleAssignment :exec
INSERT INTO role_assignments (user_id, role, merchant_id) VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING;

-- name: CreateRefreshToken :exec
INSERT INTO refresh_tokens (user_id, token_hash, family_id, expires_at, user_agent, ip)
VALUES ($1, $2, $3, $4, $5, sqlc.narg('ip')::inet);

-- name: GetRefreshTokenByHash :one
SELECT * FROM refresh_tokens WHERE token_hash = $1;

-- name: ClaimRefreshToken :one
UPDATE refresh_tokens SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL RETURNING *;

-- name: RevokeRefreshTokenByHash :exec
UPDATE refresh_tokens SET revoked_at = now() WHERE token_hash = $1 AND revoked_at IS NULL;

-- name: RevokeUserRefreshTokensExceptFamily :exec
UPDATE refresh_tokens SET revoked_at = now()
WHERE user_id = $1 AND revoked_at IS NULL AND family_id <> sqlc.arg('keep_family_id')::uuid;

-- name: RevokeAllUserRefreshTokens :exec
UPDATE refresh_tokens SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL;

