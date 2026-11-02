-- +goose Up
-- FR-01.2: business documents. Files are stored privately (never under the public media path); the
-- row keeps only metadata and the storage key. One file per kind; a new upload replaces the old one.
CREATE TABLE merchant_documents (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  merchant_id uuid NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,
  kind text NOT NULL CHECK (kind IN ('certificate', 'bank_letter')),
  storage_key text NOT NULL UNIQUE,
  content_type text NOT NULL CHECK (content_type IN ('application/pdf', 'image/jpeg', 'image/png')),
  size_bytes integer NOT NULL CHECK (size_bytes > 0),
  uploaded_by uuid NOT NULL REFERENCES users(id),
  uploaded_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (merchant_id, kind)
);

-- +goose Down
DROP TABLE IF EXISTS merchant_documents;
