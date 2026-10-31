-- +goose Up
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS unaccent;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- PostgreSQL 18 ships uuidv7(), while supported earlier versions do not. Defining
-- the same function in the application schema keeps UUIDv7 defaults portable.
-- +goose StatementBegin
CREATE FUNCTION uuidv7() RETURNS uuid
LANGUAGE plpgsql VOLATILE PARALLEL SAFE AS $$
DECLARE
  unix_millis bytea := substring(int8send(floor(extract(epoch FROM clock_timestamp()) * 1000)::bigint) FROM 3);
  random_bytes bytea := gen_random_bytes(10);
BEGIN
  random_bytes := set_byte(random_bytes, 0, (get_byte(random_bytes, 0) & 15) | 112);
  random_bytes := set_byte(random_bytes, 2, (get_byte(random_bytes, 2) & 63) | 128);
  RETURN encode(unix_millis || random_bytes, 'hex')::uuid;
END;
$$;
-- +goose StatementEnd

CREATE TYPE locale_code AS ENUM ('uz', 'ru', 'en');
CREATE TYPE currency_code AS ENUM ('UZS', 'USD');
CREATE TYPE user_status AS ENUM ('active', 'blocked');
CREATE TYPE app_role AS ENUM ('user', 'merchant_owner', 'merchant_manager', 'merchant_cashier', 'admin', 'moderator', 'support', 'finance', 'superadmin');
CREATE TYPE merchant_status AS ENUM ('active', 'suspended');
CREATE TYPE deal_kind AS ENUM ('voucher', 'product');
CREATE TYPE deal_status AS ENUM ('draft', 'pending_review', 'rejected', 'scheduled', 'active', 'paused', 'sold_out', 'expired');
CREATE TYPE redemption_method AS ENUM ('qr', 'code', 'qr_or_code');
CREATE TYPE order_status AS ENUM ('pending_payment', 'paid', 'failed', 'cancelled', 'refunded', 'partially_refunded');
CREATE TYPE fulfillment_status AS ENUM ('not_applicable', 'processing', 'shipped', 'delivered', 'returned');
CREATE TYPE payment_provider AS ENUM ('mock', 'payme', 'click', 'uzum');
CREATE TYPE payment_status AS ENUM ('created', 'pending', 'succeeded', 'failed', 'refunded');
CREATE TYPE voucher_status AS ENUM ('active', 'redeemed', 'expired', 'refunded', 'cancelled');
CREATE TYPE payout_status AS ENUM ('pending', 'processing', 'paid');
CREATE TYPE review_status AS ENUM ('published', 'hidden', 'pending');
CREATE TYPE report_status AS ENUM ('pending', 'resolved', 'dismissed');
CREATE TYPE homepage_section_type AS ENUM ('deals', 'collection', 'categories', 'banner');

-- +goose StatementBegin
CREATE FUNCTION is_localized(value jsonb, require_value boolean DEFAULT false) RETURNS boolean
LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
  SELECT jsonb_typeof(value) = 'object'
    AND value ?& ARRAY['uz', 'ru', 'en']
    AND NOT EXISTS (SELECT 1 FROM jsonb_object_keys(value) AS key WHERE key NOT IN ('uz', 'ru', 'en'))
    AND (NOT require_value OR COALESCE(value->>'uz', '') <> '' OR COALESCE(value->>'ru', '') <> '' OR COALESCE(value->>'en', '') <> '');
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION immutable_unaccent(value text) RETURNS text
LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT AS $$ SELECT public.unaccent('public.unaccent', value) $$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION set_updated_at() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN NEW.updated_at = now(); RETURN NEW; END;
$$;
-- +goose StatementEnd

CREATE TABLE users (
  id uuid PRIMARY KEY DEFAULT uuidv7(), email text NOT NULL UNIQUE, password_hash text, google_sub text UNIQUE,
  name text NOT NULL, phone text, avatar_url text, locale locale_code NOT NULL DEFAULT 'uz', currency currency_code NOT NULL DEFAULT 'UZS',
  email_verified_at timestamptz, status user_status NOT NULL DEFAULT 'active', last_login_at timestamptz, deleted_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK (password_hash IS NOT NULL OR google_sub IS NOT NULL)
);

CREATE TABLE merchants (
  id uuid PRIMARY KEY DEFAULT uuidv7(), slug text NOT NULL UNIQUE, legal_name text NOT NULL, display_name text NOT NULL,
  description jsonb NOT NULL DEFAULT '{"uz":"","ru":"","en":""}', logo_url text, cover_url text, phone text NOT NULL,
  website text, instagram text, telegram text, status merchant_status NOT NULL DEFAULT 'active', rating_avg numeric(3,2) NOT NULL DEFAULT 0,
  rating_count integer NOT NULL DEFAULT 0, commission_bps integer NOT NULL DEFAULT 1500, deleted_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK (is_localized(description)), CHECK (rating_avg BETWEEN 0 AND 5), CHECK (rating_count >= 0), CHECK (commission_bps BETWEEN 0 AND 10000)
);

CREATE TABLE role_assignments (
  id uuid PRIMARY KEY DEFAULT uuidv7(), user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE, role app_role NOT NULL,
  merchant_id uuid REFERENCES merchants(id) ON DELETE CASCADE, created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE NULLS NOT DISTINCT (user_id, role, merchant_id),
  CHECK ((role IN ('merchant_owner','merchant_manager','merchant_cashier')) = (merchant_id IS NOT NULL))
);
CREATE INDEX role_assignments_user_id_idx ON role_assignments(user_id);
CREATE INDEX role_assignments_merchant_id_idx ON role_assignments(merchant_id);

CREATE TABLE refresh_tokens (
  id uuid PRIMARY KEY DEFAULT uuidv7(), user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE, token_hash text NOT NULL UNIQUE,
  family_id uuid NOT NULL, expires_at timestamptz NOT NULL, revoked_at timestamptz, user_agent text, ip inet,
  created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX refresh_tokens_user_id_idx ON refresh_tokens(user_id);
CREATE INDEX refresh_tokens_family_id_idx ON refresh_tokens(family_id);

CREATE TABLE cities (
  id uuid PRIMARY KEY DEFAULT uuidv7(), slug text NOT NULL UNIQUE, name jsonb NOT NULL, lat numeric(9,6) NOT NULL, lng numeric(9,6) NOT NULL,
  is_active boolean NOT NULL DEFAULT true, sort_order integer NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK (is_localized(name, true)), CHECK (lat BETWEEN -90 AND 90), CHECK (lng BETWEEN -180 AND 180)
);

CREATE TABLE categories (
  id uuid PRIMARY KEY DEFAULT uuidv7(), slug text NOT NULL UNIQUE, name jsonb NOT NULL, icon text NOT NULL,
  sort_order integer NOT NULL DEFAULT 0, is_active boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), CHECK (is_localized(name, true))
);

CREATE TABLE subcategories (
  id uuid PRIMARY KEY DEFAULT uuidv7(), category_id uuid NOT NULL REFERENCES categories(id) ON DELETE CASCADE, slug text NOT NULL UNIQUE,
  name jsonb NOT NULL, sort_order integer NOT NULL DEFAULT 0, created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK (is_localized(name, true))
);
CREATE INDEX subcategories_category_id_idx ON subcategories(category_id);

CREATE TABLE merchant_locations (
  id uuid PRIMARY KEY DEFAULT uuidv7(), merchant_id uuid NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,
  city_id uuid NOT NULL REFERENCES cities(id), address text NOT NULL, lat numeric(9,6) NOT NULL, lng numeric(9,6) NOT NULL,
  opening_hours jsonb NOT NULL, phone text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK (jsonb_typeof(opening_hours) = 'object'), CHECK (lat BETWEEN -90 AND 90), CHECK (lng BETWEEN -180 AND 180)
);
CREATE INDEX merchant_locations_merchant_id_idx ON merchant_locations(merchant_id);
CREATE INDEX merchant_locations_city_id_idx ON merchant_locations(city_id);

CREATE TABLE deals (
  id uuid PRIMARY KEY DEFAULT uuidv7(), slug text NOT NULL UNIQUE, merchant_id uuid NOT NULL REFERENCES merchants(id),
  category_id uuid NOT NULL REFERENCES categories(id), subcategory_id uuid NOT NULL REFERENCES subcategories(id), kind deal_kind NOT NULL,
  title jsonb NOT NULL, short_description jsonb NOT NULL, description jsonb NOT NULL, highlights jsonb NOT NULL DEFAULT '[]',
  whats_included jsonb NOT NULL DEFAULT '[]', terms jsonb NOT NULL, faq jsonb NOT NULL DEFAULT '[]',
  redemption_method redemption_method NOT NULL DEFAULT 'qr_or_code', redemption_instructions jsonb NOT NULL,
  booking_required boolean NOT NULL DEFAULT false, is_giftable boolean NOT NULL DEFAULT false, status deal_status NOT NULL DEFAULT 'draft',
  rejection_reason text, starts_at timestamptz NOT NULL, ends_at timestamptz NOT NULL, voucher_valid_days integer,
  voucher_valid_until timestamptz, max_per_user integer NOT NULL DEFAULT 1, sold_count integer NOT NULL DEFAULT 0,
  view_count bigint NOT NULL DEFAULT 0, rating_avg numeric(3,2) NOT NULL DEFAULT 0, rating_count integer NOT NULL DEFAULT 0,
  featured_rank integer, deleted_at timestamptz, created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
  title_uz_search tsvector GENERATED ALWAYS AS (to_tsvector('simple', immutable_unaccent(COALESCE(title->>'uz','')))) STORED,
  title_ru_search tsvector GENERATED ALWAYS AS (to_tsvector('russian', immutable_unaccent(COALESCE(title->>'ru','')))) STORED,
  title_en_search tsvector GENERATED ALWAYS AS (to_tsvector('english', immutable_unaccent(COALESCE(title->>'en','')))) STORED,
  CHECK (is_localized(title, true)), CHECK (is_localized(short_description, true)), CHECK (is_localized(description, true)),
  CHECK (is_localized(terms, true)), CHECK (is_localized(redemption_instructions, true)),
  CHECK (jsonb_typeof(highlights) = 'array' AND jsonb_array_length(highlights) BETWEEN 3 AND 5),
  CHECK (jsonb_typeof(whats_included) = 'array'), CHECK (jsonb_typeof(faq) = 'array'), CHECK (starts_at < ends_at),
  CHECK ((voucher_valid_days IS NULL) <> (voucher_valid_until IS NULL)), CHECK (voucher_valid_days IS NULL OR voucher_valid_days > 0),
  CHECK (max_per_user > 0), CHECK (sold_count >= 0), CHECK (view_count >= 0), CHECK (rating_avg BETWEEN 0 AND 5), CHECK (rating_count >= 0)
);
CREATE INDEX deals_merchant_id_idx ON deals(merchant_id);
CREATE INDEX deals_category_id_idx ON deals(category_id);
CREATE INDEX deals_subcategory_id_idx ON deals(subcategory_id);
CREATE INDEX deals_status_category_ends_idx ON deals(status, category_id, ends_at);
CREATE INDEX deals_title_uz_search_idx ON deals USING gin(title_uz_search);
CREATE INDEX deals_title_ru_search_idx ON deals USING gin(title_ru_search);
CREATE INDEX deals_title_en_search_idx ON deals USING gin(title_en_search);
CREATE INDEX deals_title_uz_trgm_idx ON deals USING gin(lower(title->>'uz') gin_trgm_ops);
CREATE INDEX deals_title_ru_trgm_idx ON deals USING gin(lower(title->>'ru') gin_trgm_ops);
CREATE INDEX deals_title_en_trgm_idx ON deals USING gin(lower(title->>'en') gin_trgm_ops);

CREATE TABLE deal_locations (
  id uuid PRIMARY KEY DEFAULT uuidv7(), deal_id uuid NOT NULL REFERENCES deals(id) ON DELETE CASCADE,
  merchant_location_id uuid NOT NULL REFERENCES merchant_locations(id) ON DELETE CASCADE,
  created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), UNIQUE (deal_id, merchant_location_id)
);
CREATE INDEX deal_locations_deal_id_idx ON deal_locations(deal_id);
CREATE INDEX deal_locations_merchant_location_id_idx ON deal_locations(merchant_location_id);

CREATE TABLE deal_options (
  id uuid PRIMARY KEY DEFAULT uuidv7(), deal_id uuid NOT NULL REFERENCES deals(id) ON DELETE CASCADE, title jsonb NOT NULL,
  original_price bigint NOT NULL, deal_price bigint NOT NULL, currency currency_code NOT NULL DEFAULT 'UZS', stock_total integer NOT NULL,
  stock_sold integer NOT NULL DEFAULT 0, stock_reserved integer NOT NULL DEFAULT 0, sort_order integer NOT NULL DEFAULT 0,
  is_active boolean NOT NULL DEFAULT true, created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK (is_localized(title, true)), CHECK (currency = 'UZS'), CHECK (original_price > 0 AND deal_price > 0 AND deal_price <= original_price),
  CHECK (stock_total >= 0 AND stock_sold >= 0 AND stock_reserved >= 0 AND stock_sold + stock_reserved <= stock_total)
);
CREATE INDEX deal_options_deal_id_idx ON deal_options(deal_id);

CREATE TABLE deal_media (
  id uuid PRIMARY KEY DEFAULT uuidv7(), deal_id uuid NOT NULL REFERENCES deals(id) ON DELETE CASCADE, url text NOT NULL,
  alt jsonb NOT NULL, sort_order integer NOT NULL DEFAULT 0, width integer NOT NULL, height integer NOT NULL, blurhash text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (deal_id, sort_order), CHECK (is_localized(alt)), CHECK (width > 0 AND height > 0)
);
CREATE INDEX deal_media_deal_id_idx ON deal_media(deal_id);

CREATE TABLE collections (
  id uuid PRIMARY KEY DEFAULT uuidv7(), slug text NOT NULL UNIQUE, title jsonb NOT NULL, description jsonb NOT NULL,
  cover_url text NOT NULL, starts_at timestamptz NOT NULL, ends_at timestamptz NOT NULL, is_active boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK (is_localized(title, true)), CHECK (is_localized(description, true)), CHECK (starts_at < ends_at)
);
CREATE TABLE collection_deals (
  id uuid PRIMARY KEY DEFAULT uuidv7(), collection_id uuid NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
  deal_id uuid NOT NULL REFERENCES deals(id) ON DELETE CASCADE, sort_order integer NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), UNIQUE(collection_id, deal_id)
);
CREATE INDEX collection_deals_collection_id_idx ON collection_deals(collection_id);
CREATE INDEX collection_deals_deal_id_idx ON collection_deals(deal_id);

CREATE TABLE promo_codes (
  id uuid PRIMARY KEY DEFAULT uuidv7(), brand_name text NOT NULL, brand_logo_url text, title jsonb NOT NULL, code text,
  discount_label jsonb NOT NULL, url text NOT NULL, category_id uuid REFERENCES categories(id), starts_at timestamptz NOT NULL,
  ends_at timestamptz NOT NULL, is_active boolean NOT NULL DEFAULT true, click_count bigint NOT NULL DEFAULT 0,
  is_verified boolean NOT NULL DEFAULT false, created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK (is_localized(title, true)), CHECK (is_localized(discount_label, true)), CHECK (starts_at < ends_at), CHECK (click_count >= 0)
);
CREATE INDEX promo_codes_category_id_idx ON promo_codes(category_id);

CREATE TABLE carts (
  id uuid PRIMARY KEY DEFAULT uuidv7(), user_id uuid NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
  created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX carts_user_id_idx ON carts(user_id);
CREATE TABLE cart_items (
  id uuid PRIMARY KEY DEFAULT uuidv7(), cart_id uuid NOT NULL REFERENCES carts(id) ON DELETE CASCADE,
  deal_option_id uuid NOT NULL REFERENCES deal_options(id) ON DELETE CASCADE, quantity integer NOT NULL, is_gift boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), UNIQUE(cart_id, deal_option_id, is_gift), CHECK (quantity > 0)
);
CREATE INDEX cart_items_cart_id_idx ON cart_items(cart_id);
CREATE INDEX cart_items_deal_option_id_idx ON cart_items(deal_option_id);

CREATE TABLE orders (
  id uuid PRIMARY KEY DEFAULT uuidv7(), number text NOT NULL UNIQUE, user_id uuid NOT NULL REFERENCES users(id),
  status order_status NOT NULL DEFAULT 'pending_payment', subtotal bigint NOT NULL, total bigint NOT NULL,
  currency currency_code NOT NULL DEFAULT 'UZS', idempotency_key text NOT NULL UNIQUE, shipping_address jsonb, paid_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK (currency = 'UZS'), CHECK (subtotal >= 0 AND total >= 0 AND total <= subtotal),
  CHECK (shipping_address IS NULL OR jsonb_typeof(shipping_address) = 'object')
);
CREATE INDEX orders_user_id_created_at_idx ON orders(user_id, created_at DESC);

CREATE TABLE order_items (
  id uuid PRIMARY KEY DEFAULT uuidv7(), order_id uuid NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  deal_id uuid NOT NULL REFERENCES deals(id), deal_option_id uuid NOT NULL REFERENCES deal_options(id), title_snapshot jsonb NOT NULL,
  unit_price_snapshot bigint NOT NULL, original_price_snapshot bigint NOT NULL, quantity integer NOT NULL, is_gift boolean NOT NULL DEFAULT false,
  gift_recipient_name text, gift_recipient_email text, gift_message text, fulfillment_status fulfillment_status NOT NULL DEFAULT 'not_applicable',
  created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK (is_localized(title_snapshot, true)), CHECK (unit_price_snapshot > 0 AND original_price_snapshot >= unit_price_snapshot), CHECK (quantity > 0)
);
CREATE INDEX order_items_order_id_idx ON order_items(order_id);
CREATE INDEX order_items_deal_id_idx ON order_items(deal_id);
CREATE INDEX order_items_deal_option_id_idx ON order_items(deal_option_id);

CREATE TABLE stock_reservations (
  id uuid PRIMARY KEY DEFAULT uuidv7(), deal_option_id uuid NOT NULL REFERENCES deal_options(id), order_id uuid NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  quantity integer NOT NULL, expires_at timestamptz NOT NULL, released_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), UNIQUE(deal_option_id, order_id), CHECK(quantity > 0)
);
CREATE INDEX stock_reservations_deal_option_id_idx ON stock_reservations(deal_option_id);
CREATE INDEX stock_reservations_order_id_idx ON stock_reservations(order_id);
CREATE INDEX stock_reservations_expiry_idx ON stock_reservations(expires_at) WHERE released_at IS NULL;

CREATE TABLE payments (
  id uuid PRIMARY KEY DEFAULT uuidv7(), order_id uuid NOT NULL REFERENCES orders(id), provider payment_provider NOT NULL,
  provider_payment_id text, status payment_status NOT NULL DEFAULT 'created', amount bigint NOT NULL, currency currency_code NOT NULL DEFAULT 'UZS',
  raw_payload jsonb NOT NULL DEFAULT '{}', created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK (amount >= 0), CHECK(currency = 'UZS'), CHECK(jsonb_typeof(raw_payload) = 'object'), UNIQUE(provider, provider_payment_id)
);
CREATE INDEX payments_order_id_idx ON payments(order_id);

CREATE TABLE vouchers (
  id uuid PRIMARY KEY DEFAULT uuidv7(), order_item_id uuid NOT NULL REFERENCES order_items(id), user_id uuid NOT NULL REFERENCES users(id),
  deal_id uuid NOT NULL REFERENCES deals(id), deal_option_id uuid NOT NULL REFERENCES deal_options(id), merchant_id uuid NOT NULL REFERENCES merchants(id),
  code text NOT NULL UNIQUE, qr_token text NOT NULL UNIQUE, status voucher_status NOT NULL DEFAULT 'active', valid_from timestamptz NOT NULL,
  valid_until timestamptz NOT NULL, redeemed_at timestamptz, redeemed_by_user_id uuid REFERENCES users(id),
  redeemed_location_id uuid REFERENCES merchant_locations(id), created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK (code ~ '^[A-Z0-9]{4}-[A-Z0-9]{4}$'), CHECK (valid_from < valid_until),
  CHECK ((status = 'redeemed') = (redeemed_at IS NOT NULL)), CHECK (redeemed_at IS NULL OR redeemed_by_user_id IS NOT NULL)
);
CREATE INDEX vouchers_order_item_id_idx ON vouchers(order_item_id);
CREATE INDEX vouchers_user_id_idx ON vouchers(user_id);
CREATE INDEX vouchers_deal_id_idx ON vouchers(deal_id);
CREATE INDEX vouchers_deal_option_id_idx ON vouchers(deal_option_id);
CREATE INDEX vouchers_merchant_id_idx ON vouchers(merchant_id);
CREATE INDEX vouchers_redeemed_by_user_id_idx ON vouchers(redeemed_by_user_id);
CREATE INDEX vouchers_redeemed_location_id_idx ON vouchers(redeemed_location_id);

CREATE TABLE payouts (
  id uuid PRIMARY KEY DEFAULT uuidv7(), merchant_id uuid NOT NULL REFERENCES merchants(id), period_start date NOT NULL,
  period_end date NOT NULL, gross bigint NOT NULL, commission bigint NOT NULL, net bigint NOT NULL,
  status payout_status NOT NULL DEFAULT 'pending', paid_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(merchant_id, period_start, period_end), CHECK(period_start <= period_end), CHECK(gross >= 0 AND commission >= 0 AND net >= 0),
  CHECK(gross = commission + net), CHECK((status = 'paid') = (paid_at IS NOT NULL))
);
CREATE INDEX payouts_merchant_id_idx ON payouts(merchant_id);

CREATE TABLE reviews (
  id uuid PRIMARY KEY DEFAULT uuidv7(), deal_id uuid NOT NULL REFERENCES deals(id), merchant_id uuid NOT NULL REFERENCES merchants(id),
  user_id uuid NOT NULL REFERENCES users(id), voucher_id uuid NOT NULL UNIQUE REFERENCES vouchers(id), rating smallint NOT NULL,
  text text NOT NULL, locale locale_code NOT NULL, status review_status NOT NULL DEFAULT 'published', deleted_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), CHECK(rating BETWEEN 1 AND 5), CHECK(char_length(text) BETWEEN 20 AND 2000)
);
CREATE INDEX reviews_deal_id_idx ON reviews(deal_id);
CREATE INDEX reviews_merchant_id_idx ON reviews(merchant_id);
CREATE INDEX reviews_user_id_idx ON reviews(user_id);

CREATE TABLE review_replies (
  id uuid PRIMARY KEY DEFAULT uuidv7(), review_id uuid NOT NULL UNIQUE REFERENCES reviews(id) ON DELETE CASCADE,
  merchant_user_id uuid NOT NULL REFERENCES users(id), text text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), CHECK(char_length(text) BETWEEN 1 AND 2000)
);
CREATE INDEX review_replies_merchant_user_id_idx ON review_replies(merchant_user_id);

CREATE TABLE review_reports (
  id uuid PRIMARY KEY DEFAULT uuidv7(), review_id uuid NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
  reporter_user_id uuid NOT NULL REFERENCES users(id), reason text NOT NULL, status report_status NOT NULL DEFAULT 'pending',
  created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), UNIQUE(review_id, reporter_user_id)
);
CREATE INDEX review_reports_review_id_idx ON review_reports(review_id);
CREATE INDEX review_reports_reporter_user_id_idx ON review_reports(reporter_user_id);

CREATE TABLE saved_deals (
  id uuid PRIMARY KEY DEFAULT uuidv7(), user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  deal_id uuid NOT NULL REFERENCES deals(id) ON DELETE CASCADE, created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), UNIQUE(user_id, deal_id)
);
CREATE INDEX saved_deals_user_id_idx ON saved_deals(user_id);
CREATE INDEX saved_deals_deal_id_idx ON saved_deals(deal_id);

CREATE TABLE recently_viewed (
  id uuid PRIMARY KEY DEFAULT uuidv7(), user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  deal_id uuid NOT NULL REFERENCES deals(id) ON DELETE CASCADE, viewed_at timestamptz NOT NULL DEFAULT now(),
  created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), UNIQUE(user_id, deal_id)
);
CREATE INDEX recently_viewed_user_id_idx ON recently_viewed(user_id);
CREATE INDEX recently_viewed_deal_id_idx ON recently_viewed(deal_id);
CREATE INDEX recently_viewed_viewed_at_idx ON recently_viewed(viewed_at);

CREATE TABLE audit_logs (
  id uuid PRIMARY KEY DEFAULT uuidv7(), actor_user_id uuid REFERENCES users(id), actor_role app_role NOT NULL,
  action text NOT NULL, entity_type text NOT NULL, entity_id uuid, before_data jsonb, after_data jsonb, ip inet,
  created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK(before_data IS NULL OR jsonb_typeof(before_data) = 'object'), CHECK(after_data IS NULL OR jsonb_typeof(after_data) = 'object')
);
CREATE INDEX audit_logs_actor_user_id_idx ON audit_logs(actor_user_id);
CREATE INDEX audit_logs_entity_idx ON audit_logs(entity_type, entity_id);

CREATE TABLE exchange_rates (
  id uuid PRIMARY KEY DEFAULT uuidv7(), base currency_code NOT NULL, quote currency_code NOT NULL, rate numeric(20,8) NOT NULL,
  source text NOT NULL, effective_at timestamptz NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(base, quote, effective_at), CHECK(base <> quote), CHECK(rate > 0)
);

CREATE TABLE homepage_sections (
  id uuid PRIMARY KEY DEFAULT uuidv7(), key text NOT NULL UNIQUE, title jsonb NOT NULL, type homepage_section_type NOT NULL,
  deal_ids uuid[], rule jsonb, sort_order integer NOT NULL DEFAULT 0, is_active boolean NOT NULL DEFAULT true,
  starts_at timestamptz, ends_at timestamptz, created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK(is_localized(title, true)), CHECK(rule IS NULL OR jsonb_typeof(rule) = 'object'), CHECK(starts_at IS NULL OR ends_at IS NULL OR starts_at < ends_at)
);

CREATE TABLE banners (
  id uuid PRIMARY KEY DEFAULT uuidv7(), image_url jsonb NOT NULL, link text NOT NULL, placement text NOT NULL,
  starts_at timestamptz NOT NULL, ends_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), CHECK(is_localized(image_url, true)), CHECK(starts_at < ends_at)
);

-- +goose StatementBegin
DO $$
DECLARE table_name text;
BEGIN
  FOREACH table_name IN ARRAY ARRAY[
    'users','merchants','role_assignments','refresh_tokens','cities','categories','subcategories','merchant_locations',
    'deals','deal_locations','deal_options','deal_media','collections','collection_deals','promo_codes','carts','cart_items',
    'orders','order_items','stock_reservations','payments','vouchers','payouts','reviews','review_replies','review_reports',
    'saved_deals','recently_viewed','audit_logs','exchange_rates','homepage_sections','banners'
  ] LOOP
    EXECUTE format('CREATE TRIGGER %I_set_updated_at BEFORE UPDATE ON %I FOR EACH ROW EXECUTE FUNCTION set_updated_at()', table_name, table_name);
  END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS banners, homepage_sections, exchange_rates, audit_logs, recently_viewed, saved_deals,
  review_reports, review_replies, reviews, payouts, vouchers, payments, stock_reservations, order_items, orders,
  cart_items, carts, promo_codes, collection_deals, collections, deal_media, deal_options, deal_locations, deals,
  merchant_locations, subcategories, categories, cities, refresh_tokens, role_assignments, merchants, users CASCADE;
DROP FUNCTION IF EXISTS set_updated_at();
DROP FUNCTION IF EXISTS immutable_unaccent(text);
DROP FUNCTION IF EXISTS is_localized(jsonb, boolean);
DROP FUNCTION IF EXISTS public.uuidv7();
DROP TYPE IF EXISTS homepage_section_type, report_status, review_status, payout_status, voucher_status, payment_status,
  payment_provider, fulfillment_status, order_status, redemption_method, deal_status, deal_kind, merchant_status,
  app_role, user_status, currency_code, locale_code;
