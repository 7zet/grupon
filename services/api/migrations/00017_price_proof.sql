-- +goose Up
-- Price proof is an optional business document: a price list or menu the merchant can show to support
-- deal prices. It is not required for review, so submit counts only certificate and bank letter.
ALTER TABLE merchant_documents DROP CONSTRAINT merchant_documents_kind_check,
  ADD CONSTRAINT merchant_documents_kind_check CHECK (kind IN ('certificate', 'bank_letter', 'price_proof'));

-- +goose Down
DELETE FROM merchant_documents WHERE kind = 'price_proof';
ALTER TABLE merchant_documents DROP CONSTRAINT merchant_documents_kind_check,
  ADD CONSTRAINT merchant_documents_kind_check CHECK (kind IN ('certificate', 'bank_letter'));
