package tg

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"strings"
)

type TelegramService struct {
	bot   *tgbotapi.BotAPI
	Chats map[string]int64
}

func NewTelegramService(token string) (*TelegramService, error) {
	bot, err := tgbotapi.NewBotAPI(token)

	if err != nil {
		return nil, err
	}

	chats := map[string]int64{
		"Payments": -1003380906513,
		"Fuels":    -1003368403742,
	}

	return &TelegramService{bot, chats}, nil
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

	if strings.HasPrefix(text, "/") {
		s.handleCommand(chatID, text)
	}
}
func (s *TelegramService) handleCommand(chatID int64, text string) {
	switch text {
	case "/start":
		s.SendMessageInTelegramGroup(chatID, "hello")
	}
}
