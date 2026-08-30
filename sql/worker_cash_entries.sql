CREATE TABLE worker_cash_entries (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  worker_name VARCHAR(100) NOT NULL,
  entry_type ENUM('income', 'expense') NOT NULL,
  amount DECIMAL(12, 2) NOT NULL,
  description VARCHAR(500) NOT NULL DEFAULT '',
  entry_date DATE NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_worker_cash_worker_date (worker_name, entry_date),
  INDEX idx_worker_cash_created (created_at)
);
