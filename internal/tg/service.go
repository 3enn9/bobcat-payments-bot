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
	title := u.Message.Chat.Title

	if strings.HasPrefix(text, "/") {
		s.handleCommand(chatID, text, title)
	}
}
func (s *TelegramService) handleCommand(chatID int64, text, chatName string) {
	switch {
	case strings.HasPrefix(text, "/add"):
		s.handleAdd(chatID, text, chatName)

	}
}

func (s *TelegramService) handleAdd(chatID int64, text, title string) {
	parts := strings.Fields(text)

	if len(parts) < 3 {
		s.SendMessageInTelegramGroup(chatID, "Формат: /add описание сумма")
		return
	}

	amountStr := parts[len(parts)-1]
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		s.SendMessageInTelegramGroup(chatID, "Сумма должна быть числом")
		return
	}
	description := strings.Join(parts[1:len(parts)-1], " ")

	balance, err := s.updateBalance(chatID, title, amount)
	if err != nil {
		s.SendMessageInTelegramGroup(chatID, "Ошибка при обновлении баланса")
		log.Printf("Ошибка: %v", err)
		return
	}

	s.SendMessageInTelegramGroup(chatID,
		fmt.Sprintf(
			"💬 %s\n"+
				"💰 Сумма: %.2f\n"+
				"🏦 Касса: %.2f",
			description,
			amount,
			balance,
		),
	)
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
