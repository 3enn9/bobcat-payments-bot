-- Пользователи, которые открывали бота / miniapp
-- Выполнить вручную на VPS

CREATE TABLE bot_users (
  user_id BIGINT NOT NULL COMMENT 'MAX user id',
  name VARCHAR(255) NOT NULL DEFAULT '',
  username VARCHAR(128) NOT NULL DEFAULT '',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
