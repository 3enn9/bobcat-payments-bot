-- ID сообщения в группе водителей (DriverRequest) для обновления статуса
ALTER TABLE rogatka_requests
  ADD COLUMN driver_request_message_id VARCHAR(255) NULL AFTER max_message_id;
