-- +goose Up
ALTER TABLE order_items ADD COLUMN terms_snapshot jsonb NOT NULL
  DEFAULT '{"uz":"Shartlar","ru":"Условия","en":"Terms"}';

UPDATE order_items oi
SET terms_snapshot = d.terms
FROM deals d
WHERE d.id = oi.deal_id;

ALTER TABLE order_items
  ADD CONSTRAINT order_items_terms_snapshot_localized CHECK (is_localized(terms_snapshot, true));

-- +goose Down
ALTER TABLE order_items DROP CONSTRAINT IF EXISTS order_items_terms_snapshot_localized;
ALTER TABLE order_items DROP COLUMN IF EXISTS terms_snapshot;
