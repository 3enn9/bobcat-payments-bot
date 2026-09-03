CREATE TABLE incoming_payments (
  id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  source         ENUM('modulbank','tochka','tbank') NOT NULL,
  external_id    VARCHAR(128) NOT NULL DEFAULT '',

  executed_at    DATE         NOT NULL,
  amount         DECIMAL(14,2) NOT NULL,
  currency       VARCHAR(8)   NOT NULL DEFAULT 'RUB',

  account        VARCHAR(34)  NOT NULL DEFAULT '',
  recipient_name VARCHAR(255) NOT NULL DEFAULT '',

  payer_name     VARCHAR(255) NOT NULL DEFAULT '',
  payer_inn      VARCHAR(20)  NOT NULL DEFAULT '',

  purpose        TEXT         NOT NULL,

  raw_doc_number VARCHAR(64)  NOT NULL DEFAULT '',

  created_at     TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,

  PRIMARY KEY (id),
  UNIQUE KEY uq_incoming_source_ext (source, external_id),
  KEY idx_incoming_executed (executed_at),
  KEY idx_incoming_account  (account)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
