-- Выбранный носитель, если на карте двое через «;»
-- Выполнить вручную на VPS

ALTER TABLE fuel_entries
  ADD COLUMN holder_picked VARCHAR(255) NOT NULL DEFAULT '' COMMENT 'Кто из двух носителей заправлял' AFTER holder;
