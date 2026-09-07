-- Вид топлива / техники: petrol (бензин) или dt (ДТ)
-- Выполнить вручную на VPS

ALTER TABLE fuel_entries
  ADD COLUMN fuel_kind VARCHAR(16) NOT NULL DEFAULT '' COMMENT 'petrol или dt' AFTER equipment_number;
