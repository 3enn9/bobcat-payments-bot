-- Link invoices to incoming payments (1 payment -> N invoices).
-- Run on VPS after incoming_payments exists.

ALTER TABLE invoices
  ADD COLUMN incoming_payment_id BIGINT UNSIGNED NULL AFTER vat_amount,
  ADD KEY idx_invoices_payment (incoming_payment_id);

ALTER TABLE invoices
  ADD CONSTRAINT fk_invoices_payment
    FOREIGN KEY (incoming_payment_id) REFERENCES incoming_payments(id)
    ON DELETE SET NULL;
