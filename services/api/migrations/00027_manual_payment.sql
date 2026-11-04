-- +goose Up
ALTER TYPE payment_provider ADD VALUE IF NOT EXISTS 'manual';

-- +goose Down
-- PostgreSQL enum values are intentionally retained.
