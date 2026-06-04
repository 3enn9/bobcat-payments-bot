package tg

import (
	"PaymentsBot/internal/db"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"strings"
)

type TelegramService struct {
	bot   *tgbotapi.BotAPI
	Chats map[string]int64
	db    *db.Database
}

func NewTelegramService(token string, db *db.Database) (*TelegramService, error) {
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

	sentMsg, err := s.bot.Send(msg)
	if err != nil {
		log.Println(err)
	}

	if strings.Contains(message, "Касса:") {
		pin := tgbotapi.PinChatMessageConfig{ChatID: chatID, MessageID: sentMsg.MessageID, DisableNotification: true}
		s.bot.Request(pin)
	}
}

func (s *TelegramService) HandleUpdate(u tgbotapi.Update) {
	if u.Message == nil {
		return
	}
	text := u.Message.Text
	chatID := u.Message.Chat.ID
	chatName := u.Message.Chat.Title

	switch {
	case strings.HasPrefix(text, "/add "):
		s.handleAdd(chatID, text, chatName)
	case strings.HasPrefix(text, "/all "):
		s.handleAll(chatID)
	case strings.HasPrefix(text, "/dep "):
		s.handleDeposit(chatID, text, chatName)
	case strings.HasPrefix(text, "/salary "):
		s.handleSalary(chatID)
	}
}
