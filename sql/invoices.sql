-- Invoice tables for miniapp constructor
-- Apply manually on VPS (fresh install)

CREATE TABLE invoice_suppliers (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(255) NOT NULL,
  inn VARCHAR(20) NOT NULL,
  kpp VARCHAR(20) NOT NULL DEFAULT '',
  address_text VARCHAR(1000) NOT NULL,
  last_invoice_number INT UNSIGNED NOT NULL DEFAULT 0,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_invoice_suppliers_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE invoice_banks (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  supplier_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(255) NOT NULL,
  bik VARCHAR(20) NOT NULL,
  account VARCHAR(34) NOT NULL,
  corr_account VARCHAR(34) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_invoice_banks_name (name),
  KEY idx_invoice_banks_supplier (supplier_id),
  CONSTRAINT fk_invoice_banks_supplier
    FOREIGN KEY (supplier_id) REFERENCES invoice_suppliers(id)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE invoice_buyers (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(255) NOT NULL,
  inn VARCHAR(20) NOT NULL,
  kpp VARCHAR(20) NOT NULL DEFAULT '',
  address_text VARCHAR(1000) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_invoice_buyers_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE invoices (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  number INT UNSIGNED NOT NULL,
  invoice_date DATE NOT NULL,
  basis VARCHAR(500) NOT NULL DEFAULT '',
  supplier_id BIGINT UNSIGNED NULL,
  buyer_id BIGINT UNSIGNED NULL,
  supplier_name VARCHAR(255) NOT NULL,
  supplier_inn VARCHAR(20) NOT NULL,
  supplier_kpp VARCHAR(20) NOT NULL DEFAULT '',
  supplier_address TEXT NOT NULL,
  bank_name VARCHAR(255) NOT NULL,
  bank_bik VARCHAR(20) NOT NULL,
  bank_account VARCHAR(34) NOT NULL,
  bank_corr_account VARCHAR(34) NOT NULL,
  buyer_name VARCHAR(255) NOT NULL,
  buyer_inn VARCHAR(20) NOT NULL,
  buyer_kpp VARCHAR(20) NOT NULL DEFAULT '',
  buyer_address TEXT NOT NULL,
  total DECIMAL(14,2) NOT NULL,
  vat_amount DECIMAL(14,2) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_invoices_supplier_number (supplier_id, number),
  KEY idx_invoices_created (created_at),
  CONSTRAINT fk_invoices_supplier FOREIGN KEY (supplier_id) REFERENCES invoice_suppliers(id) ON DELETE SET NULL,
  CONSTRAINT fk_invoices_buyer FOREIGN KEY (buyer_id) REFERENCES invoice_buyers(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE invoice_items (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  invoice_id BIGINT UNSIGNED NOT NULL,
  position INT UNSIGNED NOT NULL,
  title VARCHAR(1000) NOT NULL,
  quantity DECIMAL(14,3) NOT NULL,
  unit VARCHAR(32) NOT NULL,
  price DECIMAL(14,2) NOT NULL,
  amount DECIMAL(14,2) NOT NULL,
  PRIMARY KEY (id),
  KEY idx_invoice_items_invoice (invoice_id),
  CONSTRAINT fk_invoice_items_invoice
    FOREIGN KEY (invoice_id) REFERENCES invoices(id)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO invoice_suppliers (name, inn, kpp, address_text, last_invoice_number) VALUES
('ООО "СарСтройТех"', '6454116198', '645401001',
 '410056, Саратовская обл, Саратов, В.Г. ул.им.Рахова, д. 44/54, кв. 16',
 535);

SET @supplier_id = LAST_INSERT_ID();

INSERT INTO invoice_banks (supplier_id, name, bik, account, corr_account) VALUES
(@supplier_id, 'Московский Филиал АО КБ "Модульбанк" г. Москва', '044525092', '40702810670010185610', '30101810645250000092');

INSERT INTO invoice_buyers (name, inn, kpp, address_text) VALUES
('ООО "ППС"ЛЕССТР"', '6455001697', '645501001',
 '410012, Саратовская область, г.о. город Саратов, г Саратов, ул им Слонова И.А., д. 1');
