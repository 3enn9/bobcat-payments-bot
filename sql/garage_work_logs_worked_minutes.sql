-- Добавить колонку отработанного времени (минуты)
-- Выполнить вручную на VPS, если таблица уже создана

ALTER TABLE garage_work_logs
  ADD COLUMN worked_minutes INT UNSIGNED NOT NULL DEFAULT 0
    COMMENT 'Отработано минут (time_to - time_from)'
  AFTER time_to;

-- Заполнить для уже существующих записей
UPDATE garage_work_logs
SET worked_minutes = TIMESTAMPDIFF(MINUTE, time_from, time_to)
WHERE worked_minutes = 0;
