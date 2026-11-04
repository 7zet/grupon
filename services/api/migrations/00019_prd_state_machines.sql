-- +goose Up
-- PRD 08 state machines. Values that the PRD renames or drops are mapped first, so no row loses its state.
-- Postgres cannot drop an enum value, so each affected type is rebuilt: columns move to text, the rows are
-- mapped, the type is recreated with the PRD labels, and the columns move back.
CREATE TEMP TABLE state_maps (type_name text, old_value text, new_value text);
INSERT INTO state_maps VALUES
  ('deal_status', 'pending_review', 'in_review'),
  ('deal_status', 'rejected', 'changes_requested'),
  ('deal_status', 'expired', 'ended'),
  ('deal_status', 'sold_out', 'active'),
  ('order_status', 'pending_payment', 'awaiting_payment'),
  ('payment_status', 'refunded', 'succeeded'),
  ('voucher_status', 'cancelled', 'voided');

CREATE TEMP TABLE state_labels (type_name text, label text);
INSERT INTO state_labels VALUES
  ('deal_status', 'draft'), ('deal_status', 'in_review'), ('deal_status', 'approved'), ('deal_status', 'scheduled'),
  ('deal_status', 'active'), ('deal_status', 'paused'), ('deal_status', 'ended'), ('deal_status', 'archived'),
  ('deal_status', 'changes_requested'),
  ('order_status', 'created'), ('order_status', 'awaiting_payment'), ('order_status', 'paid'), ('order_status', 'completed'),
  ('order_status', 'expired'), ('order_status', 'failed'), ('order_status', 'cancelled'), ('order_status', 'partially_refunded'),
  ('order_status', 'refunded'),
  ('payment_status', 'created'), ('payment_status', 'pending'), ('payment_status', 'succeeded'), ('payment_status', 'failed'),
  ('payment_status', 'chargeback_open'), ('payment_status', 'chargeback_won'), ('payment_status', 'chargeback_lost'),
  ('voucher_status', 'issued'), ('voucher_status', 'active'), ('voucher_status', 'partially_redeemed'),
  ('voucher_status', 'redeemed'), ('voucher_status', 'expired'), ('voucher_status', 'refunded'), ('voucher_status', 'voided');

CREATE TEMP TABLE state_columns (table_name text, column_name text, type_name text);
CREATE TEMP TABLE state_constraints (table_name text, constraint_name text, definition text);

-- +goose StatementBegin
DO $$
DECLARE
  r record;
  m record;
  t record;
BEGIN
  FOR r IN
    SELECT c.table_name, c.column_name, c.udt_name FROM information_schema.columns c
    WHERE c.table_schema = 'public' AND c.udt_name IN ('deal_status', 'order_status', 'payment_status', 'voucher_status')
  LOOP
    INSERT INTO state_columns VALUES (r.table_name, r.column_name, r.udt_name);
    EXECUTE format('ALTER TABLE %I ALTER COLUMN %I DROP DEFAULT', r.table_name, r.column_name);
  END LOOP;

  -- CHECKs that compare a state (for example vouchers: status = 'redeemed' <-> redeemed_at) are set aside.
  FOR r IN
    SELECT rel.relname AS table_name, con.conname AS constraint_name, pg_get_constraintdef(con.oid) AS definition
    FROM pg_constraint con
    JOIN pg_class rel ON rel.oid = con.conrelid
    JOIN pg_namespace ns ON ns.oid = rel.relnamespace
    WHERE ns.nspname = 'public' AND con.contype = 'c'
      AND rel.relname IN (SELECT DISTINCT table_name FROM state_columns)
      AND pg_get_constraintdef(con.oid) LIKE '%status%'
  LOOP
    INSERT INTO state_constraints VALUES (r.table_name, r.constraint_name, r.definition);
    EXECUTE format('ALTER TABLE %I DROP CONSTRAINT %I', r.table_name, r.constraint_name);
  END LOOP;

  FOR r IN SELECT table_name, column_name FROM state_columns LOOP
    EXECUTE format('ALTER TABLE %I ALTER COLUMN %I TYPE text', r.table_name, r.column_name);
  END LOOP;

  FOR m IN SELECT * FROM state_maps LOOP
    FOR r IN SELECT table_name, column_name FROM state_columns WHERE type_name = m.type_name LOOP
      EXECUTE format('UPDATE %I SET %I = %L WHERE %I = %L', r.table_name, r.column_name, m.new_value, r.column_name, m.old_value);
    END LOOP;
  END LOOP;

  FOR t IN SELECT DISTINCT type_name FROM state_columns LOOP
    EXECUTE format('DROP TYPE %I', t.type_name);
    EXECUTE format('CREATE TYPE %I AS ENUM (%s)', t.type_name,
      (SELECT string_agg(format('%L', label), ', ') FROM state_labels WHERE type_name = t.type_name));
  END LOOP;

  FOR r IN SELECT table_name, column_name, type_name FROM state_columns LOOP
    EXECUTE format('ALTER TABLE %I ALTER COLUMN %I TYPE %I USING %I::%I', r.table_name, r.column_name, r.type_name, r.column_name, r.type_name);
  END LOOP;

  FOR r IN SELECT table_name, constraint_name, definition FROM state_constraints LOOP
    EXECUTE format('ALTER TABLE %I ADD CONSTRAINT %I %s', r.table_name, r.constraint_name, r.definition);
  END LOOP;
END $$;
-- +goose StatementEnd

ALTER TABLE deals ALTER COLUMN status SET DEFAULT 'draft';
ALTER TABLE orders ALTER COLUMN status SET DEFAULT 'created';
ALTER TABLE payments ALTER COLUMN status SET DEFAULT 'created';
ALTER TABLE vouchers ALTER COLUMN status SET DEFAULT 'active';

DROP TABLE state_constraints;
DROP TABLE state_columns;
DROP TABLE state_maps;
DROP TABLE state_labels;

-- Dispute states from PRD 08. Old rows keep their state; only the allowed list grows.
ALTER TABLE disputes DROP CONSTRAINT disputes_status_check,
  ADD CONSTRAINT disputes_status_check CHECK (status IN ('open', 'awaiting_evidence', 'under_review', 'resolved_customer', 'resolved_merchant', 'closed'));

-- Voucher flags from PRD 08: a fraud hold and an open dispute each block redemption without changing the state.
ALTER TABLE vouchers ADD COLUMN fraud_hold boolean NOT NULL DEFAULT false, ADD COLUMN dispute_open boolean NOT NULL DEFAULT false;

-- The legacy payout enum has no column left after the legacy payouts table was dropped (migration 00015).
DROP TYPE IF EXISTS payout_status;

-- +goose Down
-- The state rebuild is not reversed: mapped rows keep their new labels. Restore from backup for a rollback.
ALTER TABLE vouchers DROP COLUMN IF EXISTS dispute_open, DROP COLUMN IF EXISTS fraud_hold;
