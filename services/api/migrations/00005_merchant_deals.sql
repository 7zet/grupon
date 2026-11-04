-- +goose Up
ALTER TABLE deals ADD COLUMN review_fingerprint text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE deals DROP COLUMN review_fingerprint;
