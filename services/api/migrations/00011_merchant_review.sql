-- +goose Up
-- DEALO-ALIGN step 3 (admin review of businesses, FR-01.3): reason and review time.
ALTER TABLE merchants
  ADD COLUMN rejection_reason text,
  ADD COLUMN reviewed_at timestamptz;

-- +goose Down
ALTER TABLE merchants DROP COLUMN IF EXISTS reviewed_at, DROP COLUMN IF EXISTS rejection_reason;
