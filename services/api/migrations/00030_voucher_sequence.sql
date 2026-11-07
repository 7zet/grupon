-- +goose Up
ALTER TABLE vouchers ADD COLUMN seq integer;
WITH numbered AS (
  SELECT id, row_number() OVER (PARTITION BY order_item_id ORDER BY created_at, id)::integer AS seq
  FROM vouchers
)
UPDATE vouchers v SET seq = numbered.seq FROM numbered WHERE numbered.id = v.id;
ALTER TABLE vouchers ALTER COLUMN seq SET NOT NULL;
ALTER TABLE vouchers ALTER COLUMN seq SET DEFAULT 1;
ALTER TABLE vouchers ADD CONSTRAINT vouchers_order_item_id_seq_key UNIQUE (order_item_id, seq);
ALTER TABLE vouchers ADD CONSTRAINT vouchers_seq_positive CHECK (seq > 0);

-- +goose Down
ALTER TABLE vouchers DROP CONSTRAINT vouchers_seq_positive;
ALTER TABLE vouchers DROP CONSTRAINT vouchers_order_item_id_seq_key;
ALTER TABLE vouchers DROP COLUMN seq;
