-- +goose Up
-- Single-use tokens for email verification and password reset. Only SHA-256 hashes are stored.
CREATE TABLE auth_tokens (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  kind text NOT NULL CHECK (kind IN ('verify_email', 'reset_password')),
  token_hash text NOT NULL UNIQUE,
  expires_at timestamptz NOT NULL,
  used_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX auth_tokens_user_id_idx ON auth_tokens(user_id);

-- +goose Down
DROP TABLE IF EXISTS auth_tokens;
