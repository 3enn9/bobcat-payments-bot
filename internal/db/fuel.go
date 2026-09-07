package db

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	FuelKindPetrol = "petrol"
	FuelKindDT     = "dt"
)

type FuelEntry struct {
	ID              int64   `json:"id"`
	FueledAt        string  `json:"fueledAt"`
	EquipmentNumber string  `json:"equipmentNumber"`
	FuelKind        string  `json:"fuelKind"`
	CardNumber      string  `json:"cardNumber"`
	Amount          float64 `json:"amount"`
	Holder          string  `json:"holder"`
}

type FuelEntryInput struct {
	FueledAt   time.Time
	Amount     float64
	Holder     string
	FuelKind   string
	CardNumber string
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
		INSERT INTO fuel_entries (fueled_at, fueled_date, equipment_number, fuel_kind, card_number, amount, holder)
		VALUES (?, ?, '', ?, ?, ?, ?)
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
		fuelKind := strings.TrimSpace(item.FuelKind)
		if runes := []rune(fuelKind); len(runes) > 64 {
			fuelKind = string(runes[:64])
		}
		if _, err := stmt.Exec(
			fueledAt.Format("2006-01-02 15:04:05"),
			fueledAt.Format("2006-01-02"),
			fuelKind,
			strings.TrimSpace(item.CardNumber),
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
		SELECT id, DATE_FORMAT(fueled_at, '%Y-%m-%dT%H:%i:%s'), equipment_number, fuel_kind, card_number, amount, holder
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
		if err := rows.Scan(&item.ID, &item.FueledAt, &item.EquipmentNumber, &item.FuelKind, &item.CardNumber, &item.Amount, &item.Holder); err != nil {
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

type FuelSplitPart struct {
	EquipmentNumber string  `json:"equipmentNumber"`
	Amount          float64 `json:"amount"`
}

func (d *Database) SplitFuelEntry(id int64, parts []FuelSplitPart) error {
	if id <= 0 || len(parts) < 2 {
		return fmt.Errorf("нужно минимум две части")
	}

	cleaned := make([]FuelSplitPart, 0, len(parts))
	var sumRub int64
	for _, part := range parts {
		number := strings.TrimSpace(part.EquipmentNumber)
		rub := rublesOnly(part.Amount)
		if number == "" || rub <= 0 {
			return fmt.Errorf("у каждой части укажите номер техники и сумму")
		}
		if len([]rune(number)) > 32 {
			return fmt.Errorf("номер техники слишком длинный")
		}
		cleaned = append(cleaned, FuelSplitPart{EquipmentNumber: number, Amount: float64(rub)})
		sumRub += rub
	}

	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var amount float64
	err = tx.QueryRow(`
		SELECT amount FROM fuel_entries WHERE id = ? FOR UPDATE
	`, id).Scan(&amount)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("заправка не найдена")
	}
	if err != nil {
		return err
	}
	origKop := moneyToKopecks(amount)
	if sumRub != origKop/100 {
		return fmt.Errorf("сумма частей должна быть равна %d", origKop/100)
	}

	first := cleaned[0]
	first.Amount = roundMoney(first.Amount + float64(origKop%100)/100)
	if _, err := tx.Exec(`
		UPDATE fuel_entries SET equipment_number = ?, amount = ? WHERE id = ?
	`, first.EquipmentNumber, first.Amount, id); err != nil {
		return err
	}

	for _, part := range cleaned[1:] {
		if _, err := tx.Exec(`
			INSERT INTO fuel_entries (fueled_at, fueled_date, equipment_number, fuel_kind, card_number, amount, holder)
			SELECT fueled_at, fueled_date, ?, fuel_kind, card_number, ?, holder
			FROM fuel_entries
			WHERE id = ?
		`, part.EquipmentNumber, part.Amount, id); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func moneyToKopecks(v float64) int64 {
	return int64(math.Round(v * 100))
}

func rublesOnly(v float64) int64 {
	kop := moneyToKopecks(v)
	if kop < 0 {
		return 0
	}
	return kop / 100
}

func FuelKindFromProduct(name string) string {
	n := strings.ToLower(strings.TrimSpace(name))
	n = strings.ReplaceAll(n, "ё", "е")
	switch {
	case strings.Contains(n, "dt"), strings.Contains(n, "дт"), strings.Contains(n, "дизел"):
		return FuelKindDT
	case strings.Contains(n, "бензин"), strings.Contains(n, "аи"), strings.Contains(n, "92"), strings.Contains(n, "95"), strings.Contains(n, "98"):
		return FuelKindPetrol
	default:
		return strings.TrimSpace(name)
	}
}
