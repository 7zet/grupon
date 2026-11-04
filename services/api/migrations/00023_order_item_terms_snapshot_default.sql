-- +goose Up
-- Raw administrative imports created before AT-06 may omit this field. Keep a valid conservative
-- fallback; the checkout service always supplies the deal's exact terms.
ALTER TABLE order_items ALTER COLUMN terms_snapshot
  SET DEFAULT '{"uz":"Shartlar","ru":"Условия","en":"Terms"}'::jsonb;

-- +goose Down
ALTER TABLE order_items ALTER COLUMN terms_snapshot DROP DEFAULT;
