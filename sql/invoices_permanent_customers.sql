-- Mark invoices for regular customers. Existing invoices receive the default 'нет'.

ALTER TABLE invoices
  ADD COLUMN is_regular_customer ENUM('да', 'нет') NOT NULL DEFAULT 'нет';
