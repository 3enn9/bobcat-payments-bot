-- Partial payment allocations: one payment can cover many invoices (and vice versa).
-- Run after incoming_payments / invoices exist.
-- If you already added invoices.incoming_payment_id — it is no longer used; optional drop below.

CREATE TABLE IF NOT EXISTS invoice_payment_allocations (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  payment_id BIGINT UNSIGNED NOT NULL,
  invoice_id BIGINT UNSIGNED NOT NULL,
  amount DECIMAL(14,2) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_alloc_payment_invoice (payment_id, invoice_id),
  KEY idx_alloc_invoice (invoice_id),
  CONSTRAINT fk_alloc_payment
    FOREIGN KEY (payment_id) REFERENCES incoming_payments(id)
    ON DELETE CASCADE,
  CONSTRAINT fk_alloc_invoice
    FOREIGN KEY (invoice_id) REFERENCES invoices(id)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Optional cleanup of the old 1:N column (run only if the column exists):
-- ALTER TABLE invoices DROP FOREIGN KEY fk_invoices_payment;
-- ALTER TABLE invoices DROP COLUMN incoming_payment_id;
