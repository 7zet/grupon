-- +goose Up
-- PRD §12.1: scanner lookups are counted per staff member. Failed code entries block the device after ten
-- misses; every lookup is limited to thirty a minute. Kept in PostgreSQL so it works without Redis.
CREATE TABLE redemption_attempts (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  staff_user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  failed boolean NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX redemption_attempts_staff_created_idx ON redemption_attempts(staff_user_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS redemption_attempts;
