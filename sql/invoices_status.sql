-- Invoice open/paid status for matching and manual close of historical invoices.

ALTER TABLE invoices
  ADD COLUMN status ENUM('open','paid') NOT NULL DEFAULT 'open' AFTER vat_amount,
  ADD KEY idx_invoices_status (status);
