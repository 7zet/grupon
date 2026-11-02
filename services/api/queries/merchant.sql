-- Merchant registration and lookups. Registration writes are composed in one transaction by the store.

-- name: CreateMerchant :one
INSERT INTO merchants (slug, legal_name, display_name, description, phone, stir, org_form, tax_regime,
                       bank_mfo, bank_account, responsible_name, contract_version, contract_accepted_at,
                       contract_accepted_ip)
VALUES (sqlc.arg(slug), sqlc.arg(legal_name), sqlc.arg(display_name), sqlc.arg(description), sqlc.arg(phone),
        sqlc.arg(stir), sqlc.arg(org_form), sqlc.arg(tax_regime), sqlc.arg(bank_mfo), sqlc.arg(bank_account),
        sqlc.arg(responsible_name), sqlc.arg(contract_version), sqlc.arg(contract_accepted_at),
        sqlc.narg(contract_accepted_ip))
RETURNING id, slug, legal_name, display_name, phone, status, review_status, rating_avg::float8 AS rating_avg,
          rating_count, commission_bps;

-- name: OwnedMerchantReviewStatus :one
SELECT m.id, m.review_status FROM merchants m
JOIN role_assignments ra ON ra.merchant_id = m.id
WHERE ra.user_id = sqlc.arg(user_id) AND ra.role = 'merchant_owner' AND m.deleted_at IS NULL
LIMIT 1;

-- name: SubmitMerchantForReview :one
UPDATE merchants SET review_status = 'submitted', submitted_at = sqlc.arg(at), updated_at = now()
WHERE id = sqlc.arg(id) AND review_status = 'draft'
RETURNING id, slug, legal_name, display_name, phone, status, review_status, rating_avg::float8 AS rating_avg,
          rating_count, commission_bps;

-- name: CreateMerchantLocation :one
INSERT INTO merchant_locations (merchant_id, city_id, address, lat, lng, opening_hours, phone)
VALUES ($1, $2, $3, $4, $5, '{}'::jsonb, $6)
RETURNING id;

-- name: ActiveCityExists :one
SELECT EXISTS (SELECT 1 FROM cities WHERE id = $1 AND is_active = true);

-- name: ActiveCategoryExists :one
SELECT EXISTS (SELECT 1 FROM categories WHERE id = $1 AND is_active = true);

-- name: ListMerchantsForReview :many
SELECT m.id, m.legal_name, m.display_name, m.stir, m.org_form, m.tax_regime, m.bank_mfo, m.bank_account,
       m.responsible_name, m.review_status, m.submitted_at,
       ARRAY(SELECT d.kind FROM merchant_documents d WHERE d.merchant_id = m.id ORDER BY d.kind)::text[] AS documents
FROM merchants m
WHERE m.deleted_at IS NULL AND m.review_status IN ('submitted', 'in_review')
  AND (sqlc.narg(status)::text IS NULL OR m.review_status = sqlc.narg(status)::text)
ORDER BY m.submitted_at, m.id;

-- name: MerchantReviewStatusByID :one
SELECT review_status FROM merchants WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: TransitionMerchantReview :execrows
UPDATE merchants SET review_status = sqlc.arg(to_status), rejection_reason = sqlc.narg(reason),
                     reviewed_at = now(), updated_at = now()
WHERE id = sqlc.arg(id) AND deleted_at IS NULL AND review_status = ANY(sqlc.arg(from_statuses)::text[]);

-- name: CountMerchantDocumentKinds :one
SELECT COUNT(*)::int AS kinds FROM merchant_documents WHERE merchant_id = sqlc.arg(merchant_id);

-- name: GetMerchantDocumentKey :one
SELECT storage_key FROM merchant_documents WHERE merchant_id = sqlc.arg(merchant_id) AND kind = sqlc.arg(kind);

-- name: UpsertMerchantDocument :exec
INSERT INTO merchant_documents (merchant_id, kind, storage_key, content_type, size_bytes, uploaded_by)
VALUES (sqlc.arg(merchant_id), sqlc.arg(kind), sqlc.arg(storage_key), sqlc.arg(content_type), sqlc.arg(size_bytes), sqlc.arg(uploaded_by))
ON CONFLICT (merchant_id, kind) DO UPDATE SET storage_key = EXCLUDED.storage_key, content_type = EXCLUDED.content_type,
  size_bytes = EXCLUDED.size_bytes, uploaded_by = EXCLUDED.uploaded_by, uploaded_at = now();

-- name: ListMerchantDocuments :many
SELECT kind, storage_key, content_type, size_bytes, uploaded_at FROM merchant_documents
WHERE merchant_id = sqlc.arg(merchant_id) ORDER BY kind;
