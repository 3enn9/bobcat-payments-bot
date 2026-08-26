-- Migration: banks belong to supplier (many banks per supplier)
-- Run on VPS after previous invoices.sql

ALTER TABLE invoice_banks
  ADD COLUMN supplier_id BIGINT UNSIGNED NULL AFTER id,
  ADD KEY idx_invoice_banks_supplier (supplier_id);

UPDATE invoice_banks b
INNER JOIN invoice_suppliers s ON s.bank_id = b.id
SET b.supplier_id = s.id;

-- If some banks remain without supplier, attach them to first supplier or delete manually before NOT NULL.
-- Example for orphan rows (optional):
-- UPDATE invoice_banks SET supplier_id = (SELECT id FROM invoice_suppliers ORDER BY id LIMIT 1) WHERE supplier_id IS NULL;

ALTER TABLE invoice_suppliers
  DROP FOREIGN KEY fk_invoice_suppliers_bank;

ALTER TABLE invoice_suppliers
  DROP COLUMN bank_id;

ALTER TABLE invoice_banks
  MODIFY supplier_id BIGINT UNSIGNED NOT NULL,
  ADD CONSTRAINT fk_invoice_banks_supplier
    FOREIGN KEY (supplier_id) REFERENCES invoice_suppliers(id)
    ON DELETE CASCADE;
