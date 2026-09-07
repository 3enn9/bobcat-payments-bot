-- Номер топливной карты из РН Карт
-- Выполнить вручную на VPS

ALTER TABLE fuel_entries
  ADD COLUMN card_number VARCHAR(32) NOT NULL DEFAULT '' COMMENT 'Номер топливной карты' AFTER fuel_kind;
