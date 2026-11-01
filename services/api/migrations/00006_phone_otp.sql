-- +goose Up
-- Phone sign-in: a one-time code is sent to the phone, and the account is found or created by phone.
-- Phone-only accounts keep a placeholder email until email login (R1b) is introduced.
CREATE UNIQUE INDEX users_phone_key ON users(phone) WHERE phone IS NOT NULL AND deleted_at IS NULL;
ALTER TABLE users DROP CONSTRAINT users_check;
ALTER TABLE users ADD CONSTRAINT users_sign_in_method_check
  CHECK (password_hash IS NOT NULL OR google_sub IS NOT NULL OR phone IS NOT NULL);

CREATE TABLE phone_otps (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  phone text NOT NULL,
  code_hash text NOT NULL,
  attempts integer NOT NULL DEFAULT 0,
  expires_at timestamptz NOT NULL,
  consumed_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  CHECK (attempts >= 0)
);
CREATE INDEX phone_otps_phone_created_idx ON phone_otps(phone, created_at DESC);

-- +goose Down
DROP TABLE phone_otps;
ALTER TABLE users DROP CONSTRAINT users_sign_in_method_check;
ALTER TABLE users ADD CONSTRAINT users_check CHECK (password_hash IS NOT NULL OR google_sub IS NOT NULL);
DROP INDEX users_phone_key;
