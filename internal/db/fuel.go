package db

import (
	"strings"
	"time"
)

type FuelEntry struct {
	ID              int64   `json:"id"`
	FueledAt        string  `json:"fueledAt"`
	EquipmentNumber string  `json:"equipmentNumber"`
	Amount          float64 `json:"amount"`
	Holder          string  `json:"holder"`
}

type FuelEntryInput struct {
	FueledAt time.Time
	Amount   float64
	Holder   string
}

func (d *Database) SaveFuelEntriesIfNewDate(day time.Time, entries []FuelEntryInput) (bool, error) {
	if len(entries) == 0 {
		return false, nil
	}

	tx, err := d.DB.Begin()
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()

	var count int
	if err := tx.QueryRow(`
		SELECT COUNT(*) FROM fuel_entries WHERE fueled_date = ? FOR UPDATE
	`, day.Format("2006-01-02")).Scan(&count); err != nil {
		return false, err
	}
	if count > 0 {
		return false, tx.Commit()
	}

	stmt, err := tx.Prepare(`
		INSERT INTO fuel_entries (fueled_at, fueled_date, equipment_number, amount, holder)
		VALUES (?, ?, '', ?, ?)
	`)
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	for _, item := range entries {
		fueledAt := item.FueledAt
		if fueledAt.IsZero() {
			continue
		}
		if _, err := stmt.Exec(
			fueledAt.Format("2006-01-02 15:04:05"),
			fueledAt.Format("2006-01-02"),
			item.Amount,
			strings.TrimSpace(item.Holder),
		); err != nil {
			return false, err
		}
	}

	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

func (d *Database) ListFuelEntriesByHolder(holder string, since time.Time) ([]FuelEntry, error) {
	holder = strings.TrimSpace(holder)
	if holder == "" {
		return []FuelEntry{}, nil
	}

	rows, err := d.DB.Query(`
		SELECT id, DATE_FORMAT(fueled_at, '%Y-%m-%dT%H:%i:%s'), equipment_number, amount, holder
		FROM fuel_entries
		WHERE fueled_date >= ?
		  AND holder LIKE CONCAT('%', ?, '%')
		ORDER BY fueled_at DESC, id DESC
	`, since.Format("2006-01-02"), holder)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]FuelEntry, 0)
	for rows.Next() {
		var item FuelEntry
		if err := rows.Scan(&item.ID, &item.FueledAt, &item.EquipmentNumber, &item.Amount, &item.Holder); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (d *Database) UpdateFuelEquipment(id int64, equipmentNumber string) error {
	equipmentNumber = strings.TrimSpace(equipmentNumber)
	_, err := d.DB.Exec(`
		UPDATE fuel_entries SET equipment_number = ? WHERE id = ?
	`, equipmentNumber, id)
	return err
}
