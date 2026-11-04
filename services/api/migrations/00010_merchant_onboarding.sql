-- +goose Up
-- DEALO-ALIGN step 3 (merchant onboarding, FR-01): tax and bank identity, offer acceptance and review
-- status. Existing merchants came from the seed and are treated as approved.
ALTER TABLE merchants
  ADD COLUMN stir text UNIQUE CHECK (stir ~ '^[0-9]{9}$'),
  ADD COLUMN org_form text CHECK (org_form IN ('mchj', 'yatt')),
  ADD COLUMN tax_regime text CHECK (tax_regime IN ('vat_payer', 'turnover_tax')),
  ADD COLUMN bank_mfo text CHECK (bank_mfo ~ '^[0-9]{5}$'),
  ADD COLUMN bank_account text CHECK (bank_account ~ '^[0-9]{20}$'),
  ADD COLUMN responsible_name text,
  ADD COLUMN contract_version text,
  ADD COLUMN contract_accepted_at timestamptz,
  ADD COLUMN contract_accepted_ip text,
  ADD COLUMN review_status text NOT NULL DEFAULT 'draft'
    CHECK (review_status IN ('draft', 'submitted', 'in_review', 'approved', 'rejected', 'suspended', 'terminated')),
  ADD COLUMN submitted_at timestamptz;

UPDATE merchants SET review_status = 'approved';

-- +goose Down
ALTER TABLE merchants
  DROP COLUMN IF EXISTS submitted_at, DROP COLUMN IF EXISTS review_status,
  DROP COLUMN IF EXISTS contract_accepted_ip, DROP COLUMN IF EXISTS contract_accepted_at,
  DROP COLUMN IF EXISTS contract_version, DROP COLUMN IF EXISTS responsible_name,
  DROP COLUMN IF EXISTS bank_account, DROP COLUMN IF EXISTS bank_mfo,
  DROP COLUMN IF EXISTS tax_regime, DROP COLUMN IF EXISTS org_form, DROP COLUMN IF EXISTS stir;
