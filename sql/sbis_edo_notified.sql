-- События подписания исходящих документов СБИС, уже отправленные в MAX.
CREATE TABLE IF NOT EXISTS sbis_edo_notified (
  event_key CHAR(64) NOT NULL,
  document_id VARCHAR(128) NOT NULL,
  document_number VARCHAR(255) NOT NULL DEFAULT '',
  counterparty VARCHAR(512) NOT NULL DEFAULT '',
  signed_at DATETIME NULL,
  notified_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (event_key),
  KEY idx_sbis_edo_document (document_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
