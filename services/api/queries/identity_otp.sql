-- name: GetUserByPhone :one
SELECT * FROM users WHERE phone = sqlc.arg(phone) AND deleted_at IS NULL;

-- name: CreatePhoneUser :one
INSERT INTO users (email, phone, name, locale, currency)
VALUES (sqlc.arg(email), sqlc.arg(phone), sqlc.arg(name), sqlc.arg(locale), sqlc.arg(currency))
RETURNING *;

-- name: InsertPhoneOTP :exec
INSERT INTO phone_otps (phone, code_hash, expires_at)
VALUES (sqlc.arg(phone), sqlc.arg(code_hash), sqlc.arg(expires_at));

-- name: LatestPhoneOTP :one
SELECT * FROM phone_otps
WHERE phone = sqlc.arg(phone) AND consumed_at IS NULL
ORDER BY created_at DESC, id DESC
LIMIT 1;

-- name: IncrementPhoneOTPAttempts :exec
UPDATE phone_otps SET attempts = attempts + 1 WHERE id = sqlc.arg(id);

-- name: ConsumePhoneOTP :execrows
UPDATE phone_otps SET consumed_at = now()
WHERE id = sqlc.arg(id) AND consumed_at IS NULL;

-- name: LatestIssuedPhoneOTP :one
SELECT * FROM phone_otps
WHERE phone = sqlc.arg(phone)
ORDER BY created_at DESC, id DESC
LIMIT 1;

-- name: IncrementSMSDailyUsage :one
INSERT INTO sms_daily_usage (day, scope, key, count)
VALUES (sqlc.arg(day), sqlc.arg(scope), sqlc.arg(key), 1)
ON CONFLICT (day, scope, key) DO UPDATE SET count = sms_daily_usage.count + 1
RETURNING count;
