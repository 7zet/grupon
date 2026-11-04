-- +goose Up
CREATE TABLE user_two_factor (
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    secret text NOT NULL,
    enabled_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE two_factor_recovery_codes (
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash text NOT NULL,
    used_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, code_hash)
);
CREATE TABLE two_factor_challenges (
    token_hash text PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX two_factor_challenges_user_idx ON two_factor_challenges(user_id);

-- +goose Down
DROP TABLE IF EXISTS two_factor_challenges, two_factor_recovery_codes, user_two_factor;
