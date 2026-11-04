-- +goose Up
CREATE TABLE merchant_media_assets (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  merchant_id uuid NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,
  filename text NOT NULL,
  content_type text NOT NULL CHECK (content_type IN ('image/jpeg', 'image/png', 'image/webp')),
  size bigint NOT NULL CHECK (size > 0 AND size <= 10485760),
  storage_key text NOT NULL UNIQUE,
  width integer CHECK (width IS NULL OR width > 0),
  height integer CHECK (height IS NULL OR height > 0),
  status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'ready')),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  CHECK ((status = 'ready') = (width IS NOT NULL AND height IS NOT NULL))
);
CREATE INDEX merchant_media_assets_merchant_id_idx ON merchant_media_assets(merchant_id);

-- +goose Down
DROP TABLE merchant_media_assets;
