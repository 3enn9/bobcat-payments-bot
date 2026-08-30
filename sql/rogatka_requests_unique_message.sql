-- Уникальный max_message_id для заявок Рогатка (после удаления дубликатов)
-- Выполнить вручную на VPS

-- Сначала оставить одну строку на каждый max_message_id (самую раннюю)
DELETE r1 FROM rogatka_requests r1
INNER JOIN rogatka_requests r2
  ON r1.max_message_id = r2.max_message_id AND r1.id > r2.id
WHERE r1.max_message_id IS NOT NULL AND r1.max_message_id <> '';

ALTER TABLE rogatka_requests
  ADD UNIQUE KEY uq_rogatka_requests_message (max_message_id);
