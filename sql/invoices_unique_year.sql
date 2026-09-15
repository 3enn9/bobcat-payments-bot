-- Invoice numbers restart each year: uniqueness is (supplier, number, year).

ALTER TABLE invoices
  DROP INDEX uq_invoices_supplier_number;

ALTER TABLE invoices
  ADD COLUMN invoice_year SMALLINT UNSIGNED
    AS (YEAR(invoice_date)) STORED
    AFTER invoice_date,
  ADD UNIQUE KEY uq_invoices_supplier_number_year (supplier_id, number, invoice_year);
