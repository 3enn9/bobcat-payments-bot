-- Вид топлива / техники: petrol (бензин) или dt (ДТ)
-- Выполнить вручную на VPS

ALTER TABLE fuel_entries
  ADD COLUMN fuel_kind VARCHAR(64) NOT NULL DEFAULT '' COMMENT 'Вид топлива из РН Карт (GName)' AFTER equipment_number;
