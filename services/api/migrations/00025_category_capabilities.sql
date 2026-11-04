-- +goose Up
ALTER TABLE categories RENAME COLUMN is_active TO searchable;
ALTER TABLE categories
  ADD COLUMN deal_creation_enabled boolean NOT NULL DEFAULT true,
  ADD COLUMN publish_enabled boolean NOT NULL DEFAULT true,
  ADD COLUMN purchasable boolean NOT NULL DEFAULT true,
  ADD COLUMN redeemable boolean NOT NULL DEFAULT true;
UPDATE categories
SET deal_creation_enabled = searchable,
    publish_enabled = searchable,
    purchasable = searchable,
    redeemable = true;

-- +goose Down
ALTER TABLE categories DROP COLUMN redeemable;
ALTER TABLE categories DROP COLUMN purchasable;
ALTER TABLE categories DROP COLUMN publish_enabled;
ALTER TABLE categories DROP COLUMN deal_creation_enabled;
ALTER TABLE categories RENAME COLUMN searchable TO is_active;
