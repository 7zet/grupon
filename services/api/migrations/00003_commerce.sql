-- +goose Up
CREATE SEQUENCE order_number_seq START 1;

-- request_hash fingerprints the body of the request that created an order so that a replayed
-- Idempotency-Key with a different body is rejected. Orders created before this migration have none.
ALTER TABLE orders ADD COLUMN request_hash text;

-- +goose Down
ALTER TABLE orders DROP COLUMN request_hash;
DROP SEQUENCE order_number_seq;
