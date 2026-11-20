-- Текстовые сообщения из группы «Вывозы бункера» в MAX
-- Выполнить один раз на VPS

CREATE TABLE IF NOT EXISTS bunker_hauls (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  max_chat_id BIGINT NOT NULL,
  max_message_id VARCHAR(128) NULL,
  message_date DATETIME(3) NOT NULL,
  sender_id BIGINT NOT NULL DEFAULT 0,
  sender_name VARCHAR(255) NOT NULL DEFAULT '',
  sender_username VARCHAR(128) NOT NULL DEFAULT '',
  message_text TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_bunker_hauls_message (max_chat_id, max_message_id),
  KEY idx_bunker_hauls_date (message_date),
  KEY idx_bunker_hauls_sender_date (sender_id, message_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
