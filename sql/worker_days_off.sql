-- График выходных рабочих (miniApp → подтверждение руководителем в MAX)
-- Выполнить вручную на VPS

CREATE TABLE worker_days_off (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  worker_name VARCHAR(128) NOT NULL COMMENT 'Фамилия, как в кабинете работника',
  date_from DATE NOT NULL,
  date_to DATE NOT NULL COMMENT 'Для одного дня = date_from',
  comment VARCHAR(500) NOT NULL DEFAULT '',
  status ENUM('pending', 'approved', 'rejected') NOT NULL DEFAULT 'pending',
  max_message_id VARCHAR(64) NULL COMMENT 'ID сообщения в группе Выходные для редактирования',
  max_chat_id BIGINT NULL COMMENT 'Chat ID группы (-78302034737888)',
  decided_by_user_id BIGINT NULL,
  decided_by_name VARCHAR(255) NULL,
  decided_at TIMESTAMP NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_worker_days_off_worker (worker_name),
  KEY idx_worker_days_off_dates (date_from, date_to),
  KEY idx_worker_days_off_status (status),
  KEY idx_worker_days_off_message (max_message_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
