package db

import (
	"time"
)

const bunkerHaulsSchema = `CREATE TABLE IF NOT EXISTS bunker_hauls (
	id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
	max_chat_id BIGINT NOT NULL,
	max_message_id VARCHAR(128) NULL,
	message_date DATETIME(3) NOT NULL,
	sender_id BIGINT NOT NULL DEFAULT 0,
	sender_name VARCHAR(255) NOT NULL DEFAULT '',
	sender_username VARCHAR(128) NOT NULL DEFAULT '',
	message_text TEXT NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (id),
	UNIQUE KEY uq_bunker_hauls_message (max_chat_id, max_message_id),
	KEY idx_bunker_hauls_date (message_date),
	KEY idx_bunker_hauls_sender_date (sender_id, message_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`

func (d *Database) EnsureBunkerHaulsTable() error {
	d.bunkerHaulsSchemaMu.Lock()
	defer d.bunkerHaulsSchemaMu.Unlock()

	if d.bunkerHaulsSchemaReady {
		return nil
	}
	if _, err := d.DB.Exec(bunkerHaulsSchema); err != nil {
		return err
	}
	d.bunkerHaulsSchemaReady = true
	return nil
}

func (d *Database) SaveBunkerHaulMessage(
	chatID int64,
	messageID string,
	messageDate time.Time,
	senderID int64,
	senderName string,
	senderUsername string,
	text string,
) error {
	if err := d.EnsureBunkerHaulsTable(); err != nil {
		return err
	}

	_, err := d.DB.Exec(`
		INSERT IGNORE INTO bunker_hauls
			(max_chat_id, max_message_id, message_date, sender_id, sender_name, sender_username, message_text)
		VALUES (?, NULLIF(?, ''), ?, ?, ?, ?, ?)
	`, chatID, messageID, messageDate, senderID, senderName, senderUsername, text)
	if err != nil {
		return err
	}
	return nil
}
