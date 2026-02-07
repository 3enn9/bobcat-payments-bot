package tg

import (
	"encoding/json"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"net/http"
)

func WebhookHandler(bot *TelegramService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var update tgbotapi.Update
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			log.Println("decode error", err)
			return
		}
		go bot.HandleUpdate(update)
		w.WriteHeader(http.StatusOK)
	}
}
