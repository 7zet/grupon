-- +goose Up
-- Redemptions per deal option (PRD FR-02.2): how many times one voucher of this option can be used.
ALTER TABLE deal_options ADD COLUMN redemptions_allowed integer NOT NULL DEFAULT 1 CHECK (redemptions_allowed >= 1);

-- Backup code (10 Crockford characters) lookup is by HMAC, so the code itself is never stored.
CREATE UNIQUE INDEX vouchers_zaxira_code_hmac_key ON vouchers(zaxira_code_hmac) WHERE zaxira_code_hmac IS NOT NULL;

-- Where a redemption happened, when the scanner reported it (FR-05, fraud rule for distance).
ALTER TABLE redemptions ADD COLUMN geo_lat numeric(9,6), ADD COLUMN geo_lng numeric(9,6);

-- +goose Down
ALTER TABLE redemptions DROP COLUMN geo_lng, DROP COLUMN geo_lat;
DROP INDEX vouchers_zaxira_code_hmac_key;
ALTER TABLE deal_options DROP COLUMN redemptions_allowed;
