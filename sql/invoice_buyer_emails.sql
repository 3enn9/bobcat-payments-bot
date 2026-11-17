-- Multiple email recipients for invoice buyers.

CREATE TABLE IF NOT EXISTS invoice_buyer_emails (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  buyer_id BIGINT UNSIGNED NOT NULL,
  email VARCHAR(255) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_invoice_buyer_emails_buyer_email (buyer_id, email),
  KEY idx_invoice_buyer_emails_buyer (buyer_id),
  CONSTRAINT fk_invoice_buyer_emails_buyer
    FOREIGN KEY (buyer_id) REFERENCES invoice_buyers(id)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT IGNORE INTO invoice_buyer_emails (buyer_id, email)
SELECT id, TRIM(email)
FROM invoice_buyers
WHERE TRIM(email) <> '';
