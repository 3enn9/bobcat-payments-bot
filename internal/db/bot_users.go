package db

import "strings"

func (d *Database) SaveBotUserIfNew(userID int64, name, username string) error {
	if d == nil || d.DB == nil || userID == 0 {
		return nil
	}
	_, err := d.DB.Exec(`
		INSERT IGNORE INTO bot_users (user_id, name, username)
		VALUES (?, ?, ?)
	`, userID, strings.TrimSpace(name), strings.TrimSpace(username))
	return err
}
