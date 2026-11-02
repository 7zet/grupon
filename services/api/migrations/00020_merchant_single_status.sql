-- +goose Up
-- PRD 08: a merchant has one status, review_status. The old status column (active or suspended) only
-- gated the public catalog; that gate now reads review_status = approved.
ALTER TABLE merchants DROP COLUMN status;
DROP TYPE IF EXISTS merchant_status;

-- Homepage sections keep only the deals and categories types. Rows of the removed types were deleted in
-- migration 00009, so the type can be rebuilt without a mapping.
ALTER TABLE homepage_sections ALTER COLUMN type TYPE text;
DROP TYPE homepage_section_type;
CREATE TYPE homepage_section_type AS ENUM ('deals', 'categories');
ALTER TABLE homepage_sections ALTER COLUMN type TYPE homepage_section_type USING type::homepage_section_type;

-- +goose Down
CREATE TYPE merchant_status AS ENUM ('active', 'suspended');
ALTER TABLE merchants ADD COLUMN status merchant_status NOT NULL DEFAULT 'active';
