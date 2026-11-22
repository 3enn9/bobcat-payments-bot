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

-- Ручная привязка ID отправителя MAX к фамилии водителя
CREATE TABLE IF NOT EXISTS bunker_haul_senders (
  sender_id BIGINT NOT NULL,
  surname VARCHAR(128) NOT NULL DEFAULT '',
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (sender_id),
  KEY idx_bunker_haul_senders_surname (surname)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS bunker_haul_annotations (
  haul_id BIGINT UNSIGNED NOT NULL,
  kind ENUM('vector', 'dubki', 'cash', 'base', 'rsm') NOT NULL,
  value DECIMAL(12,3) NOT NULL,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (haul_id, kind),
  CONSTRAINT fk_bunker_haul_annotations_haul FOREIGN KEY (haul_id) REFERENCES bunker_hauls(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS bunker_haul_cash_entries (
  haul_id BIGINT UNSIGNED NOT NULL,
  kind ENUM('dubki', 'cash') NOT NULL,
  cash_entry_id BIGINT UNSIGNED NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (haul_id, kind),
  UNIQUE KEY uq_bunker_haul_cash_entry (cash_entry_id),
  CONSTRAINT fk_bunker_haul_cash_entry_haul FOREIGN KEY (haul_id) REFERENCES bunker_hauls(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
