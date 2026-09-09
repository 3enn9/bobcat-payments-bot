-- Allow cancelled (annulled) invoices; they stay out of matching like paid.

ALTER TABLE invoices
  MODIFY COLUMN status ENUM('open','paid','cancelled') NOT NULL DEFAULT 'open';
