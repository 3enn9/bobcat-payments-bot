package db

import (
	"PaymentsBot/internal/domain/payment"
	"database/sql"
	"fmt"
	"time"
)

type Database struct {
	DB *sql.DB
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

func (d *Database) AddPayment(payment payment.Payment) error {
	_, err := d.DB.Exec(`
		INSERT INTO operations
		(telegram_group_id, title, operation, description, amount)
		VALUES (?, ?, ?, ?, ?)
	`,
		payment.GroupID,
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

type MiniAppRequest struct {
	ID          int64
	MaxUserID   string
	MaxUsername string
	Name        string
	Contact     string
	Message     string
	CreatedAt   time.Time
}

func (d *Database) CreateMiniAppRequest(
	maxUserID string,
	maxUsername string,
	name string,
	contact string,
	message string,
) (int64, error) {
	result, err := d.DB.Exec(`
		INSERT INTO miniapp_requests
		(max_user_id, max_username, name, contact, message)
		VALUES (?, ?, ?, ?, ?)
	`,
		maxUserID,
		maxUsername,
		name,
		contact,
		message,
	)

	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

func (d *Database) CreateRogatkaRequest(
	maxChatID int64,
	maxUserID int64,
	maxUsername string,
	maxUserName string,
	maxMessageID string,
	message string,
) (int64, error) {
	result, err := d.DB.Exec(`
		INSERT INTO rogatka_requests
		(max_chat_id, max_user_id, max_username, max_user_name, max_message_id, message)
		VALUES (?, ?, ?, ?, ?, ?)
	`,
		maxChatID,
		maxUserID,
		maxUsername,
		maxUserName,
		maxMessageID,
		message,
	)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}
