package db

import (
	"database/sql"
	"fmt"
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

const bunkerHaulAnnotationsSchema = `CREATE TABLE IF NOT EXISTS bunker_haul_annotations (
	haul_id BIGINT UNSIGNED NOT NULL,
	kind ENUM('vector', 'dubki', 'cash', 'base', 'rsm') NOT NULL,
	value DECIMAL(12,3) NOT NULL,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	PRIMARY KEY (haul_id, kind),
	CONSTRAINT fk_bunker_haul_annotations_haul FOREIGN KEY (haul_id) REFERENCES bunker_hauls(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`

const bunkerHaulCashEntriesSchema = `CREATE TABLE IF NOT EXISTS bunker_haul_cash_entries (
	haul_id BIGINT UNSIGNED NOT NULL,
	kind ENUM('dubki', 'cash') NOT NULL,
	cash_entry_id BIGINT UNSIGNED NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (haul_id, kind),
	UNIQUE KEY uq_bunker_haul_cash_entry (cash_entry_id),
	CONSTRAINT fk_bunker_haul_cash_entry_haul FOREIGN KEY (haul_id) REFERENCES bunker_hauls(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`

func (d *Database) EnsureBunkerHaulsTable() error {
	d.bunkerHaulsSchemaMu.Lock()
	defer d.bunkerHaulsSchemaMu.Unlock()

	if d.bunkerHaulsSchemaReady {
		return nil
	}
	for _, schema := range []string{bunkerHaulsSchema, bunkerHaulSendersSchema, bunkerHaulAnnotationsSchema, bunkerHaulCashEntriesSchema} {
		if _, err := d.DB.Exec(schema); err != nil {
			return err
		}
	}
	if err := d.ensureBunkerHaulAnnotationKey(); err != nil {
		return err
	}
	if err := d.ensureBunkerHaulAnnotationKinds(); err != nil {
		return err
	}
	d.bunkerHaulsSchemaReady = true
	return nil
}

func (d *Database) ensureBunkerHaulAnnotationKinds() error {
	var columnType string
	if err := d.DB.QueryRow(`
		SELECT COLUMN_TYPE
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_NAME = 'bunker_haul_annotations'
		  AND COLUMN_NAME = 'kind'
	`).Scan(&columnType); err != nil {
		return err
	}
	if strings.Contains(columnType, "'base'") && strings.Contains(columnType, "'rsm'") {
		return nil
	}
	_, err := d.DB.Exec(`
		ALTER TABLE bunker_haul_annotations
		MODIFY COLUMN kind ENUM('vector', 'dubki', 'cash', 'base', 'rsm') NOT NULL
	`)
	return err
}

func (d *Database) ensureBunkerHaulAnnotationKey() error {
	rows, err := d.DB.Query(`
		SELECT COLUMN_NAME
		FROM information_schema.KEY_COLUMN_USAGE
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_NAME = 'bunker_haul_annotations'
		  AND CONSTRAINT_NAME = 'PRIMARY'
		ORDER BY ORDINAL_POSITION
	`)
	if err != nil {
		return err
	}
	var columns []string
	for rows.Next() {
		var column string
		if err := rows.Scan(&column); err != nil {
			rows.Close()
			return err
		}
		columns = append(columns, column)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if len(columns) == 2 && columns[0] == "haul_id" && columns[1] == "kind" {
		return nil
	}
	if len(columns) != 1 || columns[0] != "haul_id" {
		return fmt.Errorf("unexpected primary key for bunker_haul_annotations: %v", columns)
	}
	_, err = d.DB.Exec(`ALTER TABLE bunker_haul_annotations DROP PRIMARY KEY, ADD PRIMARY KEY (haul_id, kind)`)
	return err
}

type BunkerHaulAnnotation struct {
	Kind   string  `json:"kind"`
	Amount float64 `json:"amount"`
}

type BunkerHaulTextMessage struct {
	ID          int64                  `json:"id"`
	SentAt      string                 `json:"sentAt"`
	Text        string                 `json:"text"`
	Annotations []BunkerHaulAnnotation `json:"annotations"`
}

func (d *Database) SuggestBunkerHaulSurnames(query string) ([]string, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []string{}, nil
	}
	if err := d.EnsureBunkerHaulsTable(); err != nil {
		return nil, err
	}

	rows, err := d.DB.Query(`
		SELECT DISTINCT TRIM(surname)
		FROM bunker_haul_senders
		WHERE TRIM(surname) <> ''
		  AND LOWER(surname) LIKE LOWER(CONCAT(?, '%'))
		ORDER BY TRIM(surname)
		LIMIT 10
	`, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	surnames := make([]string, 0)
	for rows.Next() {
		var surname string
		if err := rows.Scan(&surname); err != nil {
			return nil, err
		}
		surnames = append(surnames, surname)
	}
	return surnames, rows.Err()
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
		SELECT h.id, DATE_FORMAT(h.message_date, '%H:%i'), h.message_text,
		       a.kind, a.value
		FROM (
			SELECT h.id, h.message_date, h.message_text
			FROM bunker_hauls h
			JOIN bunker_haul_senders s ON s.sender_id = h.sender_id
			WHERE s.surname = ?
			  AND h.message_date >= ? AND h.message_date < ?
			ORDER BY h.message_date ASC, h.id ASC
			LIMIT 500
		) h
		LEFT JOIN bunker_haul_annotations a ON a.haul_id = h.id
		ORDER BY h.message_date ASC, h.id ASC, FIELD(a.kind, 'vector', 'dubki', 'cash')
	`, surname, start.Format("2006-01-02 15:04:05"), end.Format("2006-01-02 15:04:05"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]BunkerHaulTextMessage, 0)
	for rows.Next() {
		var item BunkerHaulTextMessage
		var kind sql.NullString
		var amount sql.NullFloat64
		if err := rows.Scan(&item.ID, &item.SentAt, &item.Text, &kind, &amount); err != nil {
			return nil, err
		}
		if len(items) == 0 || items[len(items)-1].ID != item.ID {
			item.Annotations = make([]BunkerHaulAnnotation, 0)
			items = append(items, item)
		}
		if kind.Valid {
			items[len(items)-1].Annotations = append(items[len(items)-1].Annotations, BunkerHaulAnnotation{Kind: kind.String, Amount: amount.Float64})
		}
	}
	return items, rows.Err()
}

func (d *Database) SaveBunkerHaulAnnotations(
	haulID int64,
	surname string,
	date time.Time,
	annotations []BunkerHaulAnnotation,
) error {
	if err := d.EnsureBunkerHaulsTable(); err != nil {
		return err
	}

	surname = strings.TrimSpace(surname)
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var text, mappedSurname string
	if err := tx.QueryRow(`
		SELECT h.message_text, s.surname
		FROM bunker_hauls h
		JOIN bunker_haul_senders s ON s.sender_id = h.sender_id
		WHERE h.id = ? FOR UPDATE
	`, haulID).Scan(&text, &mappedSurname); err != nil {
		return err
	}
	if !strings.EqualFold(surname, mappedSurname) {
		return fmt.Errorf("message does not belong to worker %q", surname)
	}

	selected := make(map[string]float64, len(annotations))
	for _, annotation := range annotations {
		if annotation.Kind != "vector" && annotation.Kind != "dubki" && annotation.Kind != "cash" && annotation.Kind != "base" && annotation.Kind != "rsm" {
			return fmt.Errorf("unknown bunker haul annotation kind %q", annotation.Kind)
		}
		if (annotation.Kind == "base" || annotation.Kind == "rsm") && annotation.Amount != 1 {
			return fmt.Errorf("checkbox annotation %q must have value 1", annotation.Kind)
		}
		if annotation.Amount <= 0 || annotation.Amount > 999999999.999 {
			return fmt.Errorf("invalid amount for bunker haul annotation %q", annotation.Kind)
		}
		if _, exists := selected[annotation.Kind]; exists {
			return fmt.Errorf("duplicate bunker haul annotation %q", annotation.Kind)
		}
		amount := annotation.Amount
		if annotation.Kind != "vector" {
			amount = float64(int64(amount*100+0.5)) / 100
		}
		selected[annotation.Kind] = amount
	}

	if _, err := tx.Exec(`DELETE FROM bunker_haul_annotations WHERE haul_id = ?`, haulID); err != nil {
		return err
	}
	for _, annotation := range annotations {
		amount := annotation.Amount
		if annotation.Kind != "vector" {
			amount = float64(int64(amount*100+0.5)) / 100
		}
		if _, err := tx.Exec(`INSERT INTO bunker_haul_annotations (haul_id, kind, value) VALUES (?, ?, ?)`, haulID, annotation.Kind, amount); err != nil {
			return err
		}
	}

	for _, kind := range []string{"dubki", "cash"} {
		amount, keep := selected[kind]
		var cashID int64
		err := tx.QueryRow(`SELECT cash_entry_id FROM bunker_haul_cash_entries WHERE haul_id = ? AND kind = ? FOR UPDATE`, haulID, kind).Scan(&cashID)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		linkExists := err == nil
		if !keep {
			if linkExists {
				if _, err := tx.Exec(`DELETE FROM worker_cash_entries WHERE id = ?`, cashID); err != nil {
					return err
				}
				if _, err := tx.Exec(`DELETE FROM bunker_haul_cash_entries WHERE haul_id = ? AND kind = ?`, haulID, kind); err != nil {
					return err
				}
			}
			continue
		}

		entryType := "income"
		if kind == "dubki" {
			entryType = "expense"
		}
		description := truncateCashDescription(strings.TrimSpace(text))
		cashEntryExists := false
		if linkExists {
			err := tx.QueryRow(`SELECT id FROM worker_cash_entries WHERE id = ? FOR UPDATE`, cashID).Scan(&cashID)
			if err == nil {
				cashEntryExists = true
			} else if err != sql.ErrNoRows {
				return err
			}
		}
		if cashEntryExists {
			if _, err := tx.Exec(`
				UPDATE worker_cash_entries
				SET worker_name = ?, entry_type = ?, amount = ?, description = ?, entry_date = ?
				WHERE id = ?
			`, surname, entryType, amount, description, date.Format("2006-01-02"), cashID); err != nil {
				return err
			}
			continue
		}
		result, err := tx.Exec(`
			INSERT INTO worker_cash_entries (worker_name, entry_type, amount, description, entry_date)
			VALUES (?, ?, ?, ?, ?)
		`, surname, entryType, amount, description, date.Format("2006-01-02"))
		if err != nil {
			return err
		}
		newCashID, err := result.LastInsertId()
		if err != nil {
			return err
		}
		if !linkExists {
			if _, err := tx.Exec(`INSERT INTO bunker_haul_cash_entries (haul_id, kind, cash_entry_id) VALUES (?, ?, ?)`, haulID, kind, newCashID); err != nil {
				return err
			}
		} else if _, err := tx.Exec(`UPDATE bunker_haul_cash_entries SET cash_entry_id = ? WHERE haul_id = ? AND kind = ?`, newCashID, haulID, kind); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func truncateCashDescription(value string) string {
	runes := []rune(value)
	if len(runes) > 500 {
		return string(runes[:500])
	}
	return value
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
