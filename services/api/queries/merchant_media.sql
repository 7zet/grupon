-- Media library: uploaded images owned by a merchant. Assets are referenced by deals through deal_media.

-- name: CreateMediaAsset :one
INSERT INTO merchant_media_assets (merchant_id, filename, content_type, size, storage_key)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, merchant_id, filename, content_type, size, storage_key, status, created_at;

-- name: GetMediaAssetForMerchant :one
SELECT id, merchant_id, filename, content_type, size, storage_key, width, height, status, created_at
FROM merchant_media_assets
WHERE id = $1 AND merchant_id = $2 AND deleted_at IS NULL;

-- name: MarkMediaAssetReady :execrows
UPDATE merchant_media_assets
SET width = $3, height = $4, status = 'ready', updated_at = now()
WHERE id = $1 AND merchant_id = $2 AND deleted_at IS NULL AND status = 'pending';

-- name: ListMediaAssets :many
SELECT id, merchant_id, filename, content_type, size, storage_key, width, height, status, created_at
FROM merchant_media_assets
WHERE merchant_id = $1 AND deleted_at IS NULL AND status = 'ready'
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: SoftDeleteUnusedMediaAsset :execrows
UPDATE merchant_media_assets a
SET deleted_at = now(), updated_at = now()
WHERE a.id = $1 AND a.merchant_id = $2 AND a.deleted_at IS NULL
  AND NOT EXISTS (SELECT 1 FROM deal_media dm WHERE dm.url LIKE '%' || a.storage_key);
