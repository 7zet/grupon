-- +goose Up
-- +goose StatementBegin
CREATE FUNCTION audit_logs_block_change() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  RAISE EXCEPTION 'audit_logs is append-only: % is not allowed', TG_OP;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER audit_logs_append_only BEFORE UPDATE OR DELETE ON audit_logs
  FOR EACH ROW EXECUTE FUNCTION audit_logs_block_change();

-- +goose Down
DROP TRIGGER audit_logs_append_only ON audit_logs;
DROP FUNCTION audit_logs_block_change();
