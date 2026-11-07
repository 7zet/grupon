-- +goose Up
CREATE TABLE sms_daily_usage (
  day date NOT NULL,
  scope text NOT NULL CHECK (scope IN ('ip', 'global')),
  key text NOT NULL,
  count integer NOT NULL DEFAULT 0 CHECK (count >= 0),
  PRIMARY KEY (day, scope, key)
);

-- +goose Down
DROP TABLE sms_daily_usage;
