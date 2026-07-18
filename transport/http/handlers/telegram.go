package handlers

import (
	"PaymentsBot/internal/tg"
	"encoding/json"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"net/http"
)

type TelegramHandler struct {
	telegram *tg.TelegramService
}

func NewTelegramHandler(useCase *tg.TelegramService) *TelegramHandler {
	return &TelegramHandler{telegram: useCase}
}

func (p *TelegramHandler) TelegramUpdates(w http.ResponseWriter, r *http.Request) {
	var update tgbotapi.Update
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		log.Println("decode error", err)
		return
	}
	err := p.telegram.Updates(update)
	if err != nil {
		log.Println(err)
	}
	w.WriteHeader(http.StatusOK)
}
