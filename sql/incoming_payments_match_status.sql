-- Hide incoming payments from matching UI without deleting them.

ALTER TABLE incoming_payments
  ADD COLUMN match_status ENUM('open','ignored') NOT NULL DEFAULT 'open' AFTER raw_doc_number,
  ADD KEY idx_incoming_match_status (match_status);
