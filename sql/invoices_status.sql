-- Invoice status for matching and manual close / annul of invoices.

ALTER TABLE invoices
  ADD COLUMN status ENUM('open','paid','cancelled') NOT NULL DEFAULT 'open' AFTER vat_amount,
  ADD KEY idx_invoices_status (status);
