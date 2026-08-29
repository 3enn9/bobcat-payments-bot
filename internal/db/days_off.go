package db

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrDaysOffNotFound = errors.New("days_off_not_found")
var ErrDaysOffAlreadyDecided = errors.New("days_off_already_decided")

type WorkerDaysOff struct {
	ID              int64     `json:"id"`
	WorkerName      string    `json:"workerName"`
	DateFrom        string    `json:"dateFrom"`
	DateTo          string    `json:"dateTo"`
	Comment         string    `json:"comment"`
	Status          string    `json:"status"`
	DecidedByName   *string   `json:"decidedByName"`
	DecidedAt       *time.Time `json:"decidedAt"`
	CreatedAt       time.Time `json:"createdAt"`
}

func (d *Database) CreateWorkerDaysOff(workerName, dateFrom, dateTo, comment string) (int64, error) {
	result, err := d.DB.Exec(`
		INSERT INTO worker_days_off (worker_name, date_from, date_to, comment, status)
		VALUES (?, ?, ?, ?, 'pending')
	`, workerName, dateFrom, dateTo, comment)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (d *Database) SetWorkerDaysOffMessage(id int64, messageID string, chatID int64) error {
	_, err := d.DB.Exec(`
		UPDATE worker_days_off
		SET max_message_id = ?, max_chat_id = ?
		WHERE id = ?
	`, messageID, chatID, id)
	return err
}

func (d *Database) ListApprovedWorkerDaysOffOnDate(date string) ([]WorkerDaysOff, error) {
	rows, err := d.DB.Query(`
		SELECT id, worker_name, date_from, date_to, comment, status,
		       decided_by_name, decided_at, created_at
		FROM worker_days_off
		WHERE status = 'approved'
		  AND date_from <= ?
		  AND date_to >= ?
		ORDER BY worker_name ASC, date_from ASC
	`, date, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]WorkerDaysOff, 0)
	for rows.Next() {
		item, err := scanWorkerDaysOff(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (d *Database) ListWorkerDaysOff(workerName, today string, limit int) ([]WorkerDaysOff, error) {
	if today == "" {
		today = time.Now().Format("2006-01-02")
	}
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	rows, err := d.DB.Query(`
		SELECT id, worker_name, date_from, date_to, comment, status,
		       decided_by_name, decided_at, created_at
		FROM worker_days_off
		WHERE worker_name = ?
		  AND date_to >= ?
		ORDER BY date_from ASC, id DESC
		LIMIT ?
	`, workerName, today, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]WorkerDaysOff, 0)
	for rows.Next() {
		item, err := scanWorkerDaysOff(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (d *Database) GetWorkerDaysOff(id int64) (*WorkerDaysOff, error) {
	row := d.DB.QueryRow(`
		SELECT id, worker_name, date_from, date_to, comment, status,
		       decided_by_name, decided_at, created_at,
		       max_message_id, max_chat_id
		FROM worker_days_off
		WHERE id = ?
	`, id)

	var item WorkerDaysOff
	var decidedBy sql.NullString
	var decidedAt sql.NullTime
	var maxMessageID sql.NullString
	var maxChatID sql.NullInt64
	var dateFrom, dateTo time.Time

	err := row.Scan(
		&item.ID, &item.WorkerName, &dateFrom, &dateTo, &item.Comment, &item.Status,
		&decidedBy, &decidedAt, &item.CreatedAt,
		&maxMessageID, &maxChatID,
	)
	if err == sql.ErrNoRows {
		return nil, ErrDaysOffNotFound
	}
	if err != nil {
		return nil, err
	}

	item.DateFrom = dateFrom.Format("2006-01-02")
	item.DateTo = dateTo.Format("2006-01-02")
	if decidedBy.Valid {
		item.DecidedByName = &decidedBy.String
	}
	if decidedAt.Valid {
		t := decidedAt.Time
		item.DecidedAt = &t
	}

	return &item, nil
}

type WorkerDaysOffRecord struct {
	WorkerDaysOff
	MaxMessageID string
	MaxChatID    int64
	OriginalText string
}

func (d *Database) GetWorkerDaysOffForDecision(id int64) (*WorkerDaysOffRecord, error) {
	row := d.DB.QueryRow(`
		SELECT id, worker_name, date_from, date_to, comment, status,
		       decided_by_name, decided_at, created_at,
		       max_message_id, max_chat_id
		FROM worker_days_off
		WHERE id = ?
	`, id)

	var item WorkerDaysOffRecord
	var decidedBy sql.NullString
	var decidedAt sql.NullTime
	var maxMessageID sql.NullString
	var maxChatID sql.NullInt64
	var dateFrom, dateTo time.Time

	err := row.Scan(
		&item.ID, &item.WorkerName, &dateFrom, &dateTo, &item.Comment, &item.Status,
		&decidedBy, &decidedAt, &item.CreatedAt,
		&maxMessageID, &maxChatID,
	)
	if err == sql.ErrNoRows {
		return nil, ErrDaysOffNotFound
	}
	if err != nil {
		return nil, err
	}

	item.DateFrom = dateFrom.Format("2006-01-02")
	item.DateTo = dateTo.Format("2006-01-02")
	if decidedBy.Valid {
		item.DecidedByName = &decidedBy.String
	}
	if decidedAt.Valid {
		t := decidedAt.Time
		item.DecidedAt = &t
	}
	if maxMessageID.Valid {
		item.MaxMessageID = maxMessageID.String
	}
	if maxChatID.Valid {
		item.MaxChatID = maxChatID.Int64
	}
	item.OriginalText = FormatDaysOffMessage(item.WorkerDaysOff)

	return &item, nil
}

func (d *Database) DecideWorkerDaysOff(id int64, status, decidedByName string, decidedByUserID int64) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var currentStatus string
	err = tx.QueryRow(`
		SELECT status FROM worker_days_off WHERE id = ? FOR UPDATE
	`, id).Scan(&currentStatus)
	if err == sql.ErrNoRows {
		return ErrDaysOffNotFound
	}
	if err != nil {
		return err
	}
	if currentStatus != "pending" {
		return ErrDaysOffAlreadyDecided
	}

	_, err = tx.Exec(`
		UPDATE worker_days_off
		SET status = ?, decided_by_user_id = ?, decided_by_name = ?, decided_at = NOW()
		WHERE id = ?
	`, status, decidedByUserID, decidedByName, id)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func scanWorkerDaysOff(rows *sql.Rows) (WorkerDaysOff, error) {
	var item WorkerDaysOff
	var decidedBy sql.NullString
	var decidedAt sql.NullTime
	var dateFrom, dateTo time.Time

	err := rows.Scan(
		&item.ID, &item.WorkerName, &dateFrom, &dateTo, &item.Comment, &item.Status,
		&decidedBy, &decidedAt, &item.CreatedAt,
	)
	if err != nil {
		return item, err
	}

	item.DateFrom = dateFrom.Format("2006-01-02")
	item.DateTo = dateTo.Format("2006-01-02")
	if decidedBy.Valid {
		item.DecidedByName = &decidedBy.String
	}
	if decidedAt.Valid {
		t := decidedAt.Time
		item.DecidedAt = &t
	}
	return item, nil
}

func FormatDaysOffMessage(item WorkerDaysOff) string {
	period := FormatDaysOffPeriod(item.DateFrom, item.DateTo)
	var b strings.Builder
	b.WriteString("Заявка на выходной\n\n")
	b.WriteString(fmt.Sprintf("Работник: %s\n", item.WorkerName))
	b.WriteString(fmt.Sprintf("Период: %s\n", period))
	if strings.TrimSpace(item.Comment) != "" {
		b.WriteString(fmt.Sprintf("Комментарий: %s\n", item.Comment))
	}
	return strings.TrimSpace(b.String())
}

func FormatDaysOffPeriod(dateFrom, dateTo string) string {
	from := formatRuDateISO(dateFrom)
	if dateFrom == dateTo {
		return from
	}
	return from + " – " + formatRuDateISO(dateTo)
}

func formatRuDateISO(iso string) string {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return iso
	}
	return fmt.Sprintf("%02d.%02d.%d", t.Day(), int(t.Month()), t.Year())
}
