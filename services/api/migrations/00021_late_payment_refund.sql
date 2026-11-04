-- +goose Up
-- AT-13: a payment that arrives after its order expired is refunded by the system. Such a refund has no
-- voucher and no requesting user, so both columns become optional and the payment is referenced instead.
ALTER TABLE refunds ALTER COLUMN voucher_id DROP NOT NULL,
  ALTER COLUMN requested_by DROP NOT NULL,
  ADD COLUMN payment_id uuid REFERENCES payments(id),
  ADD CONSTRAINT refunds_target_check CHECK (voucher_id IS NOT NULL OR payment_id IS NOT NULL);

-- +goose Down
ALTER TABLE refunds DROP CONSTRAINT refunds_target_check,
  DROP COLUMN payment_id,
  ALTER COLUMN requested_by SET NOT NULL,
  ALTER COLUMN voucher_id SET NOT NULL;
