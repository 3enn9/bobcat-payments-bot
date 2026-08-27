-- Добавить email покупателя (выполнить на VPS вручную)
ALTER TABLE invoice_buyers
  ADD COLUMN email VARCHAR(255) NOT NULL DEFAULT '' AFTER address_text;
