package main

import (
	"PaymentsBot/internal/banks"
	"PaymentsBot/internal/config"
	"PaymentsBot/internal/db"
	"PaymentsBot/internal/rncard"
	"PaymentsBot/internal/scheduler"
	"PaymentsBot/internal/tg"
	"log"
	"net/http"
)

func main() {
	cf := config.NewConfig()
	conn, err := db.NewConnectionDB(cf)
	if err != nil {
		log.Fatalf("DB connection failed: %v", err)
	}
	defer conn.Close()

	TgBotService, err := tg.NewTelegramService(cf.Token, conn)

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
