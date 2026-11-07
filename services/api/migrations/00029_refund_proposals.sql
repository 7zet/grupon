-- +goose Up
ALTER TABLE refunds ADD COLUMN proposed_by uuid REFERENCES users(id);
ALTER TABLE refunds DROP CONSTRAINT refunds_status_check;
ALTER TABLE refunds ADD CONSTRAINT refunds_status_check
  CHECK (status IN ('requested', 'proposed', 'approved', 'processing', 'succeeded', 'rejected', 'failed'));
ALTER TABLE refunds ADD CONSTRAINT refunds_proposed_by_check
  CHECK (status <> 'proposed' OR proposed_by IS NOT NULL);

-- +goose Down
ALTER TABLE refunds DROP CONSTRAINT refunds_proposed_by_check;
ALTER TABLE refunds DROP CONSTRAINT refunds_status_check;
ALTER TABLE refunds ADD CONSTRAINT refunds_status_check
  CHECK (status IN ('requested', 'approved', 'processing', 'succeeded', 'rejected', 'failed'));
ALTER TABLE refunds DROP COLUMN proposed_by;
