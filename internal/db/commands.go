package db

import (
	"PaymentsBot/internal/domain/payment"
	"database/sql"
	"fmt"
	"strings"
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

type RogatkaRequest struct {
	ID            int64     `json:"id"`
	MaxChatID     int64     `json:"maxChatId"`
	MaxUserID     int64     `json:"maxUserId"`
	MaxUsername   string    `json:"maxUsername"`
	MaxUserName   string    `json:"maxUserName"`
	MaxMessageID  string    `json:"maxMessageId"`
	Message       string    `json:"message"`
	DriverName    *string   `json:"driverName"`
	IsCompleted   bool      `json:"isCompleted"`
	DriverComment *string   `json:"driverComment"`
	CreatedAt     time.Time `json:"createdAt"`
}

func scanRogatkaRequest(scanner interface {
	Scan(dest ...any) error
}) (RogatkaRequest, error) {
	var item RogatkaRequest
	var driverName sql.NullString
	var driverComment sql.NullString
	var isCompleted int

	if err := scanner.Scan(
		&item.ID,
		&item.MaxChatID,
		&item.MaxUserID,
		&item.MaxUsername,
		&item.MaxUserName,
		&item.MaxMessageID,
		&item.Message,
		&driverName,
		&isCompleted,
		&driverComment,
		&item.CreatedAt,
	); err != nil {
		return RogatkaRequest{}, err
	}

	item.IsCompleted = isCompleted != 0
	if driverName.Valid {
		name := driverName.String
		item.DriverName = &name
	}
	if driverComment.Valid {
		comment := driverComment.String
		item.DriverComment = &comment
	}

	return item, nil
}

func (d *Database) ListRogatkaRequests(assigned bool, driverName string) ([]RogatkaRequest, error) {
	driverName = strings.TrimSpace(driverName)

	var (
		rows *sql.Rows
		err  error
	)

	switch {
	case driverName != "":
		rows, err = d.DB.Query(`
			SELECT id, max_chat_id, max_user_id, max_username, max_user_name,
			       max_message_id, message, driver_name, is_completed, driver_comment, created_at
			FROM rogatka_requests
			WHERE LOWER(driver_name) = LOWER(?)
			  AND is_completed = 0
			ORDER BY created_at DESC, id DESC
		`, driverName)
	case assigned:
		rows, err = d.DB.Query(`
			SELECT id, max_chat_id, max_user_id, max_username, max_user_name,
			       max_message_id, message, driver_name, is_completed, driver_comment, created_at
			FROM rogatka_requests
			WHERE driver_name IS NOT NULL AND driver_name <> ''
			  AND is_completed = 0
			ORDER BY created_at DESC, id DESC
		`)
	default:
		rows, err = d.DB.Query(`
			SELECT id, max_chat_id, max_user_id, max_username, max_user_name,
			       max_message_id, message, driver_name, is_completed, driver_comment, created_at
			FROM rogatka_requests
			WHERE driver_name IS NULL OR driver_name = ''
			ORDER BY created_at DESC, id DESC
		`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	requests := make([]RogatkaRequest, 0)
	for rows.Next() {
		item, err := scanRogatkaRequest(rows)
		if err != nil {
			return nil, err
		}
		requests = append(requests, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return requests, nil
}

func (d *Database) GetOpenRogatkaForDriver(id int64, driverName string) (*RogatkaRequest, error) {
	row := d.DB.QueryRow(`
		SELECT id, max_chat_id, max_user_id, max_username, max_user_name,
		       max_message_id, message, driver_name, is_completed, driver_comment, created_at
		FROM rogatka_requests
		WHERE id = ?
		  AND LOWER(driver_name) = LOWER(?)
		  AND is_completed = 0
	`, id, driverName)

	item, err := scanRogatkaRequest(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &item, nil
}

func (d *Database) CompleteRogatkaRequest(id int64, driverName, comment string) (bool, error) {
	result, err := d.DB.Exec(`
		UPDATE rogatka_requests
		SET is_completed = 1,
		    driver_comment = ?
		WHERE id = ?
		  AND LOWER(driver_name) = LOWER(?)
		  AND is_completed = 0
	`, comment, id, driverName)
	if err != nil {
		return false, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	return affected > 0, nil
}

func (d *Database) AssignRogatkaDriver(id int64, driverName string) (string, bool, error) {
	var message string
	err := d.DB.QueryRow(`
		SELECT message
		FROM rogatka_requests
		WHERE id = ?
		  AND (driver_name IS NULL OR driver_name = '')
	`, id).Scan(&message)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}

	result, err := d.DB.Exec(`
		UPDATE rogatka_requests
		SET driver_name = ?
		WHERE id = ?
		  AND (driver_name IS NULL OR driver_name = '')
	`, driverName, id)
	if err != nil {
		return "", false, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return "", false, err
	}
	if affected == 0 {
		return "", false, nil
	}

	return message, true, nil
}

func (d *Database) DeleteRogatkaRequest(id int64) (bool, error) {
	result, err := d.DB.Exec(`
		DELETE FROM rogatka_requests
		WHERE id = ?
	`, id)
	if err != nil {
		return false, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	return affected > 0, nil
}
