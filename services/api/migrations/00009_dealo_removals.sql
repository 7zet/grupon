-- +goose Up
-- DEALO-ALIGN step 2: remove features outside the pilot scope (cart, saved deals, promo codes,
-- collections, banners, exchange rate, goods, gifts, shipping, auth tokens for email flows).

DROP TABLE IF EXISTS cart_items, carts, saved_deals, collection_deals, collections, promo_codes,
  exchange_rates, banners, auth_tokens;

DELETE FROM homepage_sections WHERE type IN ('collection', 'banner');

ALTER TABLE order_items
  DROP COLUMN IF EXISTS is_gift,
  DROP COLUMN IF EXISTS gift_recipient_name,
  DROP COLUMN IF EXISTS gift_recipient_email,
  DROP COLUMN IF EXISTS gift_message,
  DROP COLUMN IF EXISTS fulfillment_status;

ALTER TABLE orders DROP COLUMN IF EXISTS shipping_address;

ALTER TABLE deals DROP COLUMN IF EXISTS kind, DROP COLUMN IF EXISTS is_giftable;

DROP TYPE IF EXISTS deal_kind;
DROP TYPE IF EXISTS fulfillment_status;

-- +goose Down
-- Data removal is not reversible. Recreate the schema from migrations 00001 and 00002 if needed.
