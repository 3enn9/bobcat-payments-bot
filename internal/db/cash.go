package db

import (
	"database/sql"
	"strings"
	"time"
)

type WorkerCashEntry struct {
	ID          int64     `json:"id"`
	WorkerName  string    `json:"workerName"`
	EntryType   string    `json:"entryType"`
	Amount      float64   `json:"amount"`
	Description string    `json:"description"`
	EntryDate   string    `json:"entryDate"`
	CreatedAt   time.Time `json:"createdAt"`
}

func (d *Database) CreateWorkerCashEntry(workerName, entryType string, amount float64, description, entryDate string) (int64, error) {
	result, err := d.DB.Exec(`
		INSERT INTO worker_cash_entries (worker_name, entry_type, amount, description, entry_date)
		VALUES (?, ?, ?, ?, ?)
	`, workerName, entryType, amount, description, entryDate)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (d *Database) ListWorkerCashEntries(workerName string, limit int) ([]WorkerCashEntry, error) {
	workerName = strings.TrimSpace(workerName)
	if limit <= 0 {
		limit = 10
	}
	rows, err := d.DB.Query(`
		SELECT id, worker_name, entry_type, amount, description,
		       DATE_FORMAT(entry_date, '%Y-%m-%d') AS entry_date, created_at
		FROM worker_cash_entries
		WHERE LOWER(worker_name) = LOWER(?)
		ORDER BY entry_date DESC, id DESC
		LIMIT ?
	`, workerName, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]WorkerCashEntry, 0)
	for rows.Next() {
		item, err := scanWorkerCashEntry(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (d *Database) CountWorkerCashEntries(workerName string) (int, error) {
	workerName = strings.TrimSpace(workerName)
	var count int
	err := d.DB.QueryRow(`
		SELECT COUNT(*)
		FROM worker_cash_entries
		WHERE LOWER(worker_name) = LOWER(?)
	`, workerName).Scan(&count)
	return count, err
}

func (d *Database) UpdateWorkerCashEntry(
	id int64,
	workerName, entryType string,
	amount float64,
	description, entryDate string,
) (bool, error) {
	result, err := d.DB.Exec(`
		UPDATE worker_cash_entries
		SET entry_type = ?, amount = ?, description = ?, entry_date = ?
		WHERE id = ?
		  AND LOWER(worker_name) = LOWER(?)
	`, entryType, amount, description, entryDate, id, workerName)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (d *Database) GetWorkerCashBalance(workerName string) (float64, error) {
	workerName = strings.TrimSpace(workerName)
	var balance sql.NullFloat64
	err := d.DB.QueryRow(`
		SELECT COALESCE(SUM(
			CASE entry_type
				WHEN 'income' THEN amount
				ELSE -amount
			END
		), 0)
		FROM worker_cash_entries
		WHERE LOWER(worker_name) = LOWER(?)
	`, workerName).Scan(&balance)
	if err != nil {
		return 0, err
	}
	if !balance.Valid {
		return 0, nil
	}
	return balance.Float64, nil
}

func scanWorkerCashEntry(scanner interface {
	Scan(dest ...any) error
}) (WorkerCashEntry, error) {
	var item WorkerCashEntry
	if err := scanner.Scan(
		&item.ID,
		&item.WorkerName,
		&item.EntryType,
		&item.Amount,
		&item.Description,
		&item.EntryDate,
		&item.CreatedAt,
	); err != nil {
		return WorkerCashEntry{}, err
	}
	return item, nil
}

func (d *Database) ListCashWorkerNames() ([]string, error) {
	rows, err := d.DB.Query(`
		SELECT name FROM (
			SELECT worker_name AS name FROM worker_cash_entries
			UNION
			SELECT worker_name AS name FROM garage_work_logs
			UNION
			SELECT worker_name AS name FROM worker_days_off
		) workers
		WHERE name IS NOT NULL AND TRIM(name) <> ''
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	names := make([]string, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}
