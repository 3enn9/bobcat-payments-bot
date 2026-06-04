package db

import (
	"database/sql"
	"fmt"
)

type Database struct {
	DB *sql.DB
}

type Payment struct {
	TelegramGroupID int64
	Title           string
	Operation       string
	Description     string
	Amount          float64
}

func (d *Database) UpdateBalance(chatID int64, title string, amount float64) error {
	_, err := d.DB.Exec(`
		INSERT INTO workers (chat_id, title, balance)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE 
			balance = balance + VALUES(balance),
			title = VALUES(title)
	`, chatID, title, amount)
	if err != nil {
		return err
	}
	return nil
}

func (d *Database) GetBalance(chatID int64) (float64, error) {
	var balance float64
	err := d.DB.QueryRow(`
		SELECT balance FROM workers WHERE chat_id = ?
	`, chatID).Scan(&balance)
	if err != nil {
		return 0, err
	}

	return balance, nil
}

func (d *Database) AllBalance() (string, error) {

	rows, err := d.DB.Query(`
		SELECT title, balance 
		FROM workers 
		ORDER BY title
	`)

	defer rows.Close()
	var result string

	for rows.Next() {
		var title string
		var balance float64

		if err := rows.Scan(&title, &balance); err != nil {
			fmt.Printf("error")
		}
		if title == "" {
			continue
		}
		result += fmt.Sprintf("• %s — %.2f\n", title, balance)
	}

	return result, err
}

func (d *Database) AddPayment(payment Payment) error {
	_, err := d.DB.Exec(`
		INSERT INTO operations
		(telegram_group_id, title, operation, description, amount)
		VALUES (?, ?, ?, ?, ?)
	`,
		payment.TelegramGroupID,
		payment.Title,
		payment.Operation,
		payment.Description,
		payment.Amount,
	)

	if err != nil {
		return err
	}

	return nil
}
