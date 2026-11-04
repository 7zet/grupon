-- name: UserIDByPhone :one
SELECT id FROM users WHERE phone = sqlc.arg(phone) AND deleted_at IS NULL;

-- name: ListMerchantStaffRows :many
SELECT ra.id, ra.role::text AS role, u.id AS user_id, u.name, u.email, u.phone, u.locale, u.currency, u.status
FROM role_assignments ra
JOIN users u ON u.id = ra.user_id AND u.deleted_at IS NULL
WHERE ra.merchant_id = sqlc.arg(merchant_id) AND ra.role::text IN ('merchant_owner', 'merchant_manager', 'merchant_cashier')
ORDER BY ra.created_at, ra.id;

-- name: InsertStaffAssignment :one
INSERT INTO role_assignments (user_id, role, merchant_id)
VALUES (sqlc.arg(user_id), sqlc.arg(role)::app_role, sqlc.arg(merchant_id))
RETURNING id;

-- name: StaffAssignmentForMerchant :one
SELECT id, user_id, role::text AS role FROM role_assignments
WHERE id = sqlc.arg(id) AND merchant_id = sqlc.arg(merchant_id);

-- name: UpdateStaffRole :execrows
UPDATE role_assignments SET role = sqlc.arg(role)::app_role, updated_at = now()
WHERE id = sqlc.arg(id) AND merchant_id = sqlc.arg(merchant_id) AND role::text <> 'merchant_owner';

-- name: DeleteStaffAssignment :execrows
DELETE FROM role_assignments
WHERE id = sqlc.arg(id) AND merchant_id = sqlc.arg(merchant_id) AND role::text <> 'merchant_owner';
