package main

import (
	"PaymentsBot/internal/banks"
	"PaymentsBot/internal/rncard"
	"PaymentsBot/internal/scheduler"
	"PaymentsBot/internal/tg"
	"log"
	"net/http"
)

func main() {
	TgBotService, err := tg.NewTelegramService("8440241939:AAEvMsPT9FeOFWlvexZfvmxg9GcOxXoR7yE")

	if err != nil {
		log.Fatalf("error create tgbot %v", err)
	}

	banks.SetTelegram(TgBotService)
	rncard.SetTelegram(TgBotService)

	scheduler.SendDailyScheduler(rncard.FetchAndSendTransactions)

	http.HandleFunc("/telegram/webhook", tg.WebhookHandler(TgBotService))
	http.HandleFunc("/webhook", banks.TochkaBankHandler)
	http.HandleFunc("/modulbank", banks.ModuleBankHandler)
	http.HandleFunc("/tbank", banks.TBankHandler)
	log.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
