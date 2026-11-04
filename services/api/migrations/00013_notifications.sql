-- +goose Up
-- PRD FR-05.4: buyers learn about each redemption at once and can report it as "not me". Notifications
-- are in-app for now; SMS and Telegram come with their providers.
CREATE TABLE notifications (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  kind text NOT NULL CHECK (kind IN ('voucher_redeemed')),
  voucher_id uuid REFERENCES vouchers(id) ON DELETE CASCADE,
  merchant_name text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  read_at timestamptz
);
CREATE INDEX notifications_user_created_idx ON notifications(user_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS notifications;
