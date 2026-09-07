-- Заправки РН Карт (ежедневный импорт + номер техники вручную)
-- Выполнить вручную на VPS

CREATE TABLE fuel_entries (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  fueled_at DATETIME NOT NULL COMMENT 'Дата и время заправки',
  fueled_date DATE NOT NULL COMMENT 'Дата заправки (для проверки «день уже импортирован»)',
  equipment_number VARCHAR(32) NOT NULL DEFAULT '' COMMENT 'Номер техники, которую заправил рабочий',
  fuel_kind VARCHAR(16) NOT NULL DEFAULT '' COMMENT 'petrol (бензин) или dt (ДТ)',
  amount DECIMAL(14,2) NOT NULL COMMENT 'Сумма',
  holder VARCHAR(255) NOT NULL DEFAULT '' COMMENT 'Носитель карты',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_fuel_date (fueled_date),
  KEY idx_fuel_holder (holder)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
