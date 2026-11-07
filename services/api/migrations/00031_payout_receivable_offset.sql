-- +goose Up
ALTER TABLE payout_runs ADD COLUMN receivable_offset bigint NOT NULL DEFAULT 0 CHECK (receivable_offset >= 0);

-- +goose Down
ALTER TABLE payout_runs DROP COLUMN receivable_offset;
