-- +goose Up
ALTER TYPE app_role ADD VALUE IF NOT EXISTS 'system';
ALTER TABLE disputes DROP CONSTRAINT disputes_status_check;
ALTER TABLE disputes ADD CONSTRAINT disputes_status_check CHECK (status IN (
  'open', 'awaiting_evidence', 'under_review', 'resolved_customer', 'resolved_merchant', 'closed'
));

-- +goose Down
-- PostgreSQL enum values are not removed on downgrade; `system` remains reserved for audit actors.
UPDATE disputes SET status = 'resolved_merchant' WHERE status IN ('awaiting_evidence', 'under_review', 'closed');
ALTER TABLE disputes DROP CONSTRAINT disputes_status_check;
ALTER TABLE disputes ADD CONSTRAINT disputes_status_check CHECK (status IN ('open', 'resolved_customer', 'resolved_merchant'));
