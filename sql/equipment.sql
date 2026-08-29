-- Справочник техники (номера)
-- Выполнить вручную на VPS

CREATE TABLE equipment (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  number VARCHAR(16) NOT NULL COMMENT 'Номер техники',
  is_active TINYINT(1) NOT NULL DEFAULT 1,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_equipment_number (number),
  KEY idx_equipment_active (is_active)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO equipment (number) VALUES
('2995'),
('9860'),
('1739'),
('0077'),
('1707'),
('2068'),
('0152'),
('2972'),
('4865'),
('3025'),
('0142'),
('6262'),
('0145'),
('1995'),
('571'),
('512'),
('604'),
('798'),
('352'),
('676'),
('475'),
('899'),
('701'),
('088'),
('108'),
('504'),
('436'),
('187'),
('394'),
('459'),
('3092'),
('3091'),
('0049'),
('4464');
