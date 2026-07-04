package main

import (
	"PaymentsBot/internal/banks"
	"PaymentsBot/internal/config"
	"PaymentsBot/internal/db"
	multi "PaymentsBot/internal/multiMessenger"
	"PaymentsBot/internal/rncard"
	"PaymentsBot/internal/scheduler"
	"PaymentsBot/internal/tg"
	"PaymentsBot/internal/usecase"
	"PaymentsBot/transport"
	"PaymentsBot/transport/http/handlers"
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cf := config.NewConfig()

	dbInstance, err := db.NewConnectionDB(cf)
	if err != nil {
		log.Fatalf("DB connection failed: %v", err)
	}
	defer dbInstance.DB.Close()

	tgBotService, err := tg.NewTelegramService(cf.Token, dbInstance)
	if err != nil {
		log.Fatalf("error create tgbot %v", err)
	}

	multiMessengers := multi.NewMultiMessenger([]usecase.SendMessanger{tgBotService})
	rnCardService := rncard.NewRnCardService(multiMessengers)
	banksService := banks.NewBankService(multiMessengers)
	banksHandler := handlers.NewBanksHandler(banksService)
	telegramHandler := handlers.NewTelegramHandler(tgBotService)
	router := transport.NewRouter(banksHandler, telegramHandler)

	scheduler.SendDailyScheduler(rnCardService.FetchAndSendTransactions)

	server := &http.Server{Handler: router, Addr: ":8080"}

	go func() {
		log.Println("Server started at :8080")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)

	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	<-stop

	log.Println("Shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)

	defer cancel()

	err = tgBotService.SendMessageInGroupID(877804669, "Server stopped")
	if err != nil {
		log.Printf("error send server stopped message: %v", err)
	}

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}

	if err := dbInstance.DB.Close(); err != nil {
		log.Printf("db close error: %v", err)
	}

	log.Println("Server stopped")
}
