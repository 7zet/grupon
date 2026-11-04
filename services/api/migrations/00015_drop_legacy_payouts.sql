-- +goose Up
-- The first payouts table is replaced by payout_runs (migration 00007). Nothing reads it any more.
DROP TABLE IF EXISTS payouts;
DROP TYPE IF EXISTS payout_status;

-- +goose Down
-- Data removal is not reversible.
