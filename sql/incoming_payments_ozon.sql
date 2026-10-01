-- Add Ozon bank as incoming payment source.

ALTER TABLE incoming_payments
  MODIFY COLUMN source ENUM('modulbank','tochka','tbank','ozon') NOT NULL;
