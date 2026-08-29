-- Журнал работ в гараже (miniApp → Гараж)
-- Выполнить вручную на VPS

CREATE TABLE garage_work_logs (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  worker_name VARCHAR(128) NOT NULL COMMENT 'Фамилия',
  work_date DATE NOT NULL COMMENT 'Дата работы',
  time_from TIME NOT NULL COMMENT 'Время начала работы',
  time_to TIME NOT NULL COMMENT 'Время окончания работы',
  worked_minutes INT UNSIGNED NOT NULL COMMENT 'Отработано минут (time_to - time_from)',
  description TEXT NOT NULL COMMENT 'Что сделал за время работы',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_garage_work_worker (worker_name),
  KEY idx_garage_work_date (work_date),
  KEY idx_garage_work_worker_date (worker_name, work_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
