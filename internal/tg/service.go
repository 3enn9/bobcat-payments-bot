package tg

import (
	"database/sql"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"strconv"
	"strings"
)

type TelegramService struct {
	bot   *tgbotapi.BotAPI
	Chats map[string]int64
	db    *sql.DB
}

func NewTelegramService(token string, db *sql.DB) (*TelegramService, error) {
	bot, err := tgbotapi.NewBotAPI(token)

	if err != nil {
		return nil, err
	}

	chats := map[string]int64{
		"Payments": -1003380906513,
		"Fuels":    -1003368403742,
		"Cash":     -1003797529492,
	}

	return &TelegramService{bot, chats, db}, nil
}

func (s *TelegramService) SendMessageInTelegramGroup(chatID int64, message string) {

	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = "HTML"

	_, err := s.bot.Send(msg)
	if err != nil {
		log.Println(err)
	}
}

func (s *TelegramService) HandleUpdate(u tgbotapi.Update) {
	if u.Message == nil {
		return
	}
	text := u.Message.Text
	chatID := u.Message.Chat.ID
	chatName := u.Message.Chat.Title

	if strings.HasPrefix(text, "/") {
		s.handleCommand(chatID, text, chatName)
	}

}
func (s *TelegramService) handleCommand(chatID int64, text, chatName string) {
	switch {
	case strings.HasPrefix(text, "/add"):
		s.handleAdd(chatID, text, chatName)
	case strings.HasPrefix(text, "/all"):
		s.handleAll(chatID)
	}
}

func (s *TelegramService) handleAdd(chatID int64, text, chatName string) {

	operationArray := strings.Split(text, "\n")

	var operations []string
	var errorsList []string
	var totalAmount float64
	var balance float64

	for i, operation := range operationArray {
		parts := strings.Fields(operation)

		if len(parts) < 3 {
			s.SendMessageInTelegramGroup(chatID, "Формат: /add описание сумма\nописание сумма\n...")
			return
		}

		if i == 0 {
			parts = parts[1:]
		}

		amountStr := parts[len(parts)-1]
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			errorsList = append(errorsList, operation+" (неверная сумма)")
			continue
		}
		description := strings.Join(parts[:len(parts)-1], " ")

		balance, err = s.updateBalance(chatID, chatName, amount)
		if err != nil {
			errorsList = append(errorsList, operation+" (ошибка БД)")
			continue
		}

		operations = append(operations, fmt.Sprintf("• %s: %.2f", description, amount))
		totalAmount += amount

		if amount > 0 {
			s.SendMessageInTelegramGroup(s.Chats["Cash"],
				fmt.Sprintf(
					"💬 %s\n💰 Сумма: %.2f\n",
					description,
					amount,
				),
			)
		}
	}
	msg := fmt.Sprintf(
		"📊 Операции:\n%s\n\n💰 Итого: %.2f\n🏦 Касса: %.2f",
		strings.Join(operations, "\n"),
		totalAmount,
		balance,
	)

	if len(errorsList) > 0 {
		msg += "\n\n⚠ Пропущены:\n" + strings.Join(errorsList, "\n")
	}

	s.SendMessageInTelegramGroup(chatID, msg)
}

func (s *TelegramService) updateBalance(chatID int64, title string, amount float64) (float64, error) {
	_, err := s.db.Exec(`
		INSERT INTO workers (chat_id, title, balance)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE 
			balance = balance + VALUES(balance),
			title = VALUES(title)
	`, chatID, title, amount)
	if err != nil {
		return 0, err
	}

	// получаем текущий баланс
	var balance float64
	err = s.db.QueryRow(`
		SELECT balance FROM workers WHERE chat_id = ?
	`, chatID).Scan(&balance)
	if err != nil {
		return 0, err
	}

	return balance, nil
}

func (s *TelegramService) handleAll(chatID int64) {
	rows, err := s.db.Query(`
		SELECT title, balance 
		FROM workers 
		ORDER BY title
	`)
	if err != nil {
		s.SendMessageInTelegramGroup(chatID, err.Error())
		return
	}
	defer rows.Close()

	var result string
	for rows.Next() {
		var title string
		var balance float64

		if err := rows.Scan(&title, &balance); err != nil {
			s.SendMessageInTelegramGroup(chatID, err.Error())
			return
		}
		if title == "" {
			continue
		}

		result += fmt.Sprintf("• %s — %.2f\n", title, balance)

	}

	if result == "" {
		result = "Нет данных"
	}
	s.SendMessageInTelegramGroup(chatID, result)
}
