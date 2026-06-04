package main

import (
	"PaymentsBot/internal/banks"
	"PaymentsBot/internal/config"
	"PaymentsBot/internal/db"
	"PaymentsBot/internal/rncard"
	"PaymentsBot/internal/scheduler"
	"PaymentsBot/internal/tg"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cf := config.NewConfig()
	DBInstance, err := db.NewConnectionDB(cf)
	if err != nil {
		log.Fatalf("DB connection failed: %v", err)
	}
	defer DBInstance.DB.Close()

	TgBotService, err := tg.NewTelegramService(cf.Token, DBInstance)

	if err != nil {
		log.Fatalf("error create tgbot %v", err)
	}

	banks.SetTelegram(TgBotService)
	rncard.SetTelegram(TgBotService)

	scheduler.SendDailyScheduler(rncard.FetchAndSendTransactions)
	mux := http.NewServeMux()

	server := &http.Server{Handler: mux, Addr: ":8080"}

	mux.HandleFunc("/telegram/webhook", tg.WebhookHandler(TgBotService))
	mux.HandleFunc("/webhook", banks.TochkaBankHandler)
	mux.HandleFunc("/modulbank", banks.ModuleBankHandler)
	mux.HandleFunc("/tbank", banks.TBankHandler)

	go func() {
		log.Println("Server started at :8080")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)

	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	<-stop

	log.Println("Shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)

	defer cancel()

	TgBotService.SendMessageInTelegramGroup(877804669, "Server stopped")

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}

	if err := DBInstance.DB.Close(); err != nil {
		log.Printf("db close error: %v", err)
	}

	log.Println("Server stopped")

}
