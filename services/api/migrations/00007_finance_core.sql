-- +goose Up
-- Finance core: append-only double-entry ledger, redemptions with multi-use counts, payouts with
-- four-eyes approval, refunds, disputes, payout holds, fraud signals, outbox and reconciliation.
-- Money is bigint tiyin (ADR-0003). Every journal entry balances, enforced at commit.

-- Vouchers can be multi-use (BR-03): redemptions_used never exceeds redemptions_allowed.
ALTER TABLE vouchers
  ADD COLUMN redemptions_allowed integer NOT NULL DEFAULT 1 CHECK (redemptions_allowed >= 1),
  ADD COLUMN redemptions_used integer NOT NULL DEFAULT 0,
  ADD CONSTRAINT vouchers_redemptions_within_allowed CHECK (redemptions_used >= 0 AND redemptions_used <= redemptions_allowed),
  ADD COLUMN zaxira_code_hmac text;

CREATE TABLE redemptions (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  voucher_id uuid NOT NULL REFERENCES vouchers(id),
  merchant_id uuid NOT NULL REFERENCES merchants(id),
  merchant_location_id uuid NOT NULL REFERENCES merchant_locations(id),
  staff_user_id uuid NOT NULL REFERENCES users(id),
  sequence integer NOT NULL CHECK (sequence >= 1),
  amount bigint NOT NULL CHECK (amount > 0),
  commission bigint NOT NULL CHECK (commission >= 0 AND commission <= amount),
  created_at timestamptz NOT NULL DEFAULT now(),
  reversed_at timestamptz,
  reversed_by_user_id uuid REFERENCES users(id),
  reversal_reason text,
  payout_run_id uuid,
  UNIQUE (voucher_id, sequence),
  CHECK ((reversed_at IS NULL) = (reversed_by_user_id IS NULL))
);
CREATE INDEX redemptions_merchant_created_idx ON redemptions(merchant_id, created_at);
CREATE INDEX redemptions_staff_created_idx ON redemptions(staff_user_id, created_at);

-- Ledger accounts, one row per account code such as customer_prepaid:{merchant}.
CREATE TABLE ledger_accounts (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  code text NOT NULL UNIQUE,
  kind text NOT NULL CHECK (kind IN ('asset', 'liability', 'revenue', 'expense')),
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE journal_entries (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  kind text NOT NULL,
  reference text NOT NULL,
  idempotency_key text NOT NULL UNIQUE,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE journal_lines (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  entry_id uuid NOT NULL REFERENCES journal_entries(id),
  account_id uuid NOT NULL REFERENCES ledger_accounts(id),
  direction text NOT NULL CHECK (direction IN ('debit', 'credit')),
  amount bigint NOT NULL CHECK (amount > 0)
);
CREATE INDEX journal_lines_entry_idx ON journal_lines(entry_id);
CREATE INDEX journal_lines_account_idx ON journal_lines(account_id);

-- Ledger rows are never changed or removed (BR-09).
-- +goose StatementBegin
CREATE FUNCTION ledger_block_change() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  RAISE EXCEPTION 'ledger is append-only: % on % is not allowed', TG_OP, TG_TABLE_NAME;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER journal_entries_append_only BEFORE UPDATE OR DELETE ON journal_entries
  FOR EACH ROW EXECUTE FUNCTION ledger_block_change();
CREATE TRIGGER journal_lines_append_only BEFORE UPDATE OR DELETE ON journal_lines
  FOR EACH ROW EXECUTE FUNCTION ledger_block_change();

-- Each entry needs at least two lines and equal debit and credit totals (BR-10). Checked at commit,
-- so an entry can be written line by line inside one transaction.
-- +goose StatementBegin
CREATE FUNCTION ledger_check_balance() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
  debits bigint;
  credits bigint;
  lines integer;
BEGIN
  SELECT count(*),
         COALESCE(sum(amount) FILTER (WHERE direction = 'debit'), 0),
         COALESCE(sum(amount) FILTER (WHERE direction = 'credit'), 0)
    INTO lines, debits, credits
    FROM journal_lines WHERE entry_id = NEW.entry_id;
  IF lines < 2 OR debits <> credits THEN
    RAISE EXCEPTION 'journal entry % is not balanced (lines %, debit %, credit %)', NEW.entry_id, lines, debits, credits;
  END IF;
  RETURN NULL;
END;
$$;
-- +goose StatementEnd
CREATE CONSTRAINT TRIGGER journal_lines_balanced AFTER INSERT ON journal_lines
  DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION ledger_check_balance();

CREATE TABLE payout_runs (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  merchant_id uuid NOT NULL REFERENCES merchants(id),
  amount bigint NOT NULL CHECK (amount > 0),
  status text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'pending_approval', 'approved', 'sent', 'reconciled', 'failed')),
  prepared_by uuid NOT NULL REFERENCES users(id),
  approved_by uuid REFERENCES users(id),
  bank_doc_ref text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  -- Four-eyes (AT-18): the person who prepared a payout cannot approve it.
  CHECK (approved_by IS NULL OR approved_by <> prepared_by)
);
ALTER TABLE redemptions ADD CONSTRAINT redemptions_payout_run_fk FOREIGN KEY (payout_run_id) REFERENCES payout_runs(id);

CREATE TABLE refunds (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  voucher_id uuid NOT NULL REFERENCES vouchers(id),
  amount bigint NOT NULL CHECK (amount > 0),
  reason text NOT NULL,
  status text NOT NULL DEFAULT 'requested' CHECK (status IN ('requested', 'approved', 'processing', 'succeeded', 'rejected', 'failed')),
  requested_by uuid NOT NULL REFERENCES users(id),
  approved_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK (approved_by IS NULL OR approved_by <> requested_by)
);

CREATE TABLE disputes (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  redemption_id uuid NOT NULL UNIQUE REFERENCES redemptions(id),
  opened_by uuid NOT NULL REFERENCES users(id),
  reason text NOT NULL,
  status text NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'resolved_customer', 'resolved_merchant')),
  created_at timestamptz NOT NULL DEFAULT now(),
  resolved_at timestamptz
);

-- A merchant on hold cannot receive payouts (BR-11 bank change, fraud review).
CREATE TABLE merchant_payout_holds (
  merchant_id uuid PRIMARY KEY REFERENCES merchants(id),
  reason text NOT NULL,
  hold_until timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE fraud_signals (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  merchant_id uuid REFERENCES merchants(id),
  subject_user_id uuid REFERENCES users(id),
  rule text NOT NULL,
  detail jsonb NOT NULL DEFAULT '{}',
  status text NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed')),
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE outbox (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  topic text NOT NULL,
  payload jsonb NOT NULL DEFAULT '{}',
  created_at timestamptz NOT NULL DEFAULT now(),
  published_at timestamptz
);

CREATE TABLE reconciliation_findings (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  check_name text NOT NULL,
  expected bigint NOT NULL,
  actual bigint NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE reconciliation_findings;
DROP TABLE outbox;
DROP TABLE fraud_signals;
DROP TABLE merchant_payout_holds;
DROP TABLE disputes;
DROP TABLE refunds;
ALTER TABLE redemptions DROP CONSTRAINT redemptions_payout_run_fk;
DROP TABLE payout_runs;
DROP TRIGGER journal_lines_balanced ON journal_lines;
DROP TRIGGER journal_lines_append_only ON journal_lines;
DROP TRIGGER journal_entries_append_only ON journal_entries;
DROP FUNCTION ledger_check_balance();
DROP FUNCTION ledger_block_change();
DROP TABLE journal_lines;
DROP TABLE journal_entries;
DROP TABLE ledger_accounts;
DROP TABLE redemptions;
ALTER TABLE vouchers
  DROP CONSTRAINT vouchers_redemptions_within_allowed,
  DROP COLUMN zaxira_code_hmac,
  DROP COLUMN redemptions_used,
  DROP COLUMN redemptions_allowed;
