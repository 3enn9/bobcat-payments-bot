package db

import (
	"sort"
	"strconv"
	"strings"
)

type Equipment struct {
	ID     int64  `json:"id"`
	Number string `json:"number"`
}

func (d *Database) ListEquipment() ([]Equipment, error) {
	rows, err := d.DB.Query(`
		SELECT id, number
		FROM equipment
		WHERE is_active = 1
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]Equipment, 0)
	for rows.Next() {
		var item Equipment
		if err := rows.Scan(&item.ID, &item.Number); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	sort.Slice(result, func(i, j int) bool {
		return compareEquipmentNumber(result[i].Number, result[j].Number)
	})

	return result, nil
}

func compareEquipmentNumber(a, b string) bool {
	ai, errA := strconv.ParseInt(strings.TrimLeft(a, "0"), 10, 64)
	bi, errB := strconv.ParseInt(strings.TrimLeft(b, "0"), 10, 64)
	if errA == nil && errB == nil {
		if ai != bi {
			return ai < bi
		}
	}
	if len(a) != len(b) {
		return len(a) < len(b)
	}
	return a < b
}
