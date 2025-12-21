package tg

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
)

func SendMessageInTelegramGroup(message string) {
	bot, err := tgbotapi.NewBotAPI("8440241939:AAEvMsPT9FeOFWlvexZfvmxg9GcOxXoR7yE")

	if err != nil {
		log.Panic(err)
	}

	chatID := int64(-1003380906513)
	msg := tgbotapi.NewMessage(chatID, message)

	_, err = bot.Send(msg)
	if err != nil {
		log.Panic(err)
	}
	log.Println("Сообщение отправлено")
}
