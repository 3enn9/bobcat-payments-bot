package tg

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
)

type TelegramService struct {
	bot   *tgbotapi.BotAPI
	chats map[string]int64
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

func (s *TelegramService) SendMessageInTelegramGroup(chatType, message string) {
	chatID, ok := s.chats[chatType]
	if !ok {
		log.Printf("Unknown key %s", chatType)
		return
	}

	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = "HTML"

	_, err := s.bot.Send(msg)
	if err != nil {
		log.Println(err)
	}
}
