-- Cursor of last processed IMAP UID for accountant invoice emails.
-- Apply on VPS. Without this table the importer will not start.

CREATE TABLE IF NOT EXISTS invoice_mail_cursor (
  mailbox VARCHAR(64) NOT NULL,
  last_uid BIGINT UNSIGNED NOT NULL DEFAULT 0,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (mailbox)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
