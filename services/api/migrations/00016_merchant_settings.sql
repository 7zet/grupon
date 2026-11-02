-- +goose Up
-- Merchant operational and notification settings (contract: MerchantSettings).
ALTER TABLE merchants
  ADD COLUMN order_emails boolean NOT NULL DEFAULT true,
  ADD COLUMN review_emails boolean NOT NULL DEFAULT true,
  ADD COLUMN payout_emails boolean NOT NULL DEFAULT true,
  ADD COLUMN default_location_id uuid REFERENCES merchant_locations(id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE merchants
  DROP COLUMN IF EXISTS default_location_id, DROP COLUMN IF EXISTS payout_emails,
  DROP COLUMN IF EXISTS review_emails, DROP COLUMN IF EXISTS order_emails;
