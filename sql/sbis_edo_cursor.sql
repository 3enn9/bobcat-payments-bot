-- Последнее обработанное событие СБИС для восстановления после перерывов связи.
CREATE TABLE IF NOT EXISTS sbis_edo_cursor (
  cursor_id TINYINT NOT NULL PRIMARY KEY,
  event_datetime VARCHAR(19) NOT NULL,
  event_id VARCHAR(128) NOT NULL DEFAULT '',
  document_id VARCHAR(128) NOT NULL DEFAULT ''
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
