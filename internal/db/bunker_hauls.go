package db

import (
	"strings"
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

const bunkerHaulSendersSchema = `CREATE TABLE IF NOT EXISTS bunker_haul_senders (
	sender_id BIGINT NOT NULL,
	surname VARCHAR(128) NOT NULL DEFAULT '',
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	PRIMARY KEY (sender_id),
	KEY idx_bunker_haul_senders_surname (surname)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`

func (d *Database) EnsureBunkerHaulsTable() error {
	d.bunkerHaulsSchemaMu.Lock()
	defer d.bunkerHaulsSchemaMu.Unlock()

	if d.bunkerHaulsSchemaReady {
		return nil
	}
	for _, schema := range []string{bunkerHaulsSchema, bunkerHaulSendersSchema} {
		if _, err := d.DB.Exec(schema); err != nil {
			return err
		}
	}
	d.bunkerHaulsSchemaReady = true
	return nil
}

type BunkerHaulTextMessage struct {
	ID     int64  `json:"id"`
	SentAt string `json:"sentAt"`
	Text   string `json:"text"`
}

func (d *Database) ListBunkerHaulMessagesBySurname(surname string, date time.Time) ([]BunkerHaulTextMessage, error) {
	surname = strings.TrimSpace(surname)
	if surname == "" {
		return []BunkerHaulTextMessage{}, nil
	}
	if err := d.EnsureBunkerHaulsTable(); err != nil {
		return nil, err
	}

	start := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	end := start.AddDate(0, 0, 1)
	rows, err := d.DB.Query(`
		SELECT h.id, DATE_FORMAT(h.message_date, '%H:%i'), h.message_text
		FROM bunker_hauls h
		JOIN bunker_haul_senders s ON s.sender_id = h.sender_id
		WHERE s.surname = ?
		  AND h.message_date >= ? AND h.message_date < ?
		ORDER BY h.message_date ASC, h.id ASC
		LIMIT 500
	`, surname, start.Format("2006-01-02 15:04:05"), end.Format("2006-01-02 15:04:05"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]BunkerHaulTextMessage, 0)
	for rows.Next() {
		var item BunkerHaulTextMessage
		if err := rows.Scan(&item.ID, &item.SentAt, &item.Text); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
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
