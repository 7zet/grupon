-- +goose Up
-- Login is by phone code only (PRD). Password and Google sign-in are gone, so their columns go too.
ALTER TABLE users DROP COLUMN password_hash, DROP COLUMN google_sub;

-- Money is UZS only. Postgres cannot drop an enum value, so the type is rebuilt. Every currency column
-- moves to text, its CHECK constraints (for example CHECK (currency = 'UZS')) are set aside, the type is
-- rebuilt with UZS alone, and columns and constraints move back. A column still holding USD stops the
-- migration: relabelling money silently would be wrong, so someone must decide what those rows mean first.
CREATE TEMP TABLE currency_columns (table_name text, column_name text);
CREATE TEMP TABLE currency_constraints (table_name text, constraint_name text, definition text);

-- +goose StatementBegin
DO $$
DECLARE
  r record;
  usd_rows bigint;
BEGIN
  FOR r IN
    SELECT c.table_name, c.column_name FROM information_schema.columns c
    WHERE c.table_schema = 'public' AND c.udt_name = 'currency_code'
  LOOP
    EXECUTE format('SELECT count(*) FROM %I WHERE %I::text = ''USD''', r.table_name, r.column_name) INTO usd_rows;
    IF usd_rows > 0 THEN
      RAISE EXCEPTION 'cannot drop USD: % row(s) in %.%', usd_rows, r.table_name, r.column_name;
    END IF;
    INSERT INTO currency_columns VALUES (r.table_name, r.column_name);
  END LOOP;

  FOR r IN
    SELECT rel.relname AS table_name, con.conname AS constraint_name, pg_get_constraintdef(con.oid) AS definition
    FROM pg_constraint con
    JOIN pg_class rel ON rel.oid = con.conrelid
    JOIN pg_namespace ns ON ns.oid = rel.relnamespace
    WHERE ns.nspname = 'public' AND con.contype = 'c' AND pg_get_constraintdef(con.oid) LIKE '%currency%'
  LOOP
    INSERT INTO currency_constraints VALUES (r.table_name, r.constraint_name, r.definition);
    EXECUTE format('ALTER TABLE %I DROP CONSTRAINT %I', r.table_name, r.constraint_name);
  END LOOP;

  FOR r IN SELECT table_name, column_name FROM currency_columns LOOP
    EXECUTE format('ALTER TABLE %I ALTER COLUMN %I DROP DEFAULT', r.table_name, r.column_name);
    EXECUTE format('ALTER TABLE %I ALTER COLUMN %I TYPE text', r.table_name, r.column_name);
  END LOOP;
END $$;
-- +goose StatementEnd

DROP TYPE currency_code;
CREATE TYPE currency_code AS ENUM ('UZS');

-- +goose StatementBegin
DO $$
DECLARE
  r record;
BEGIN
  FOR r IN SELECT table_name, column_name FROM currency_columns LOOP
    EXECUTE format('ALTER TABLE %I ALTER COLUMN %I TYPE currency_code USING %I::currency_code', r.table_name, r.column_name, r.column_name);
    EXECUTE format('ALTER TABLE %I ALTER COLUMN %I SET DEFAULT ''UZS''', r.table_name, r.column_name);
  END LOOP;
  FOR r IN SELECT table_name, constraint_name, definition FROM currency_constraints LOOP
    EXECUTE format('ALTER TABLE %I ADD CONSTRAINT %I %s', r.table_name, r.constraint_name, r.definition);
  END LOOP;
END $$;
-- +goose StatementEnd

DROP TABLE currency_columns;
DROP TABLE currency_constraints;

-- +goose Down
ALTER TABLE users ADD COLUMN password_hash text, ADD COLUMN google_sub text UNIQUE;
ALTER TYPE currency_code ADD VALUE 'USD';
