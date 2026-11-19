package main

import (
	"PaymentsBot/internal/accountant"
	"PaymentsBot/internal/banks"
	"PaymentsBot/internal/clock"
	"PaymentsBot/internal/codexagent"
	"PaymentsBot/internal/config"
	"PaymentsBot/internal/db"
	mailpkg "PaymentsBot/internal/mail"
	max2 "PaymentsBot/internal/max"
	multi "PaymentsBot/internal/multiMessenger"
	"PaymentsBot/internal/payments"
	"PaymentsBot/internal/rncard"
	"PaymentsBot/internal/sbis"
	"PaymentsBot/internal/scheduler"
	"PaymentsBot/internal/tg"
	"PaymentsBot/internal/usecase"
	"PaymentsBot/internal/webhookqueue"
	"PaymentsBot/transport"
	"PaymentsBot/transport/http/handlers"
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func main() {
	cf := config.NewConfig()
	clock.Init(cf.Timezone)

	dbInstance, err := db.NewConnectionDB(cf)
	if err != nil {
		log.Fatalf("DB connection failed: %v", err)
	}
	defer dbInstance.DB.Close()
	incomingWebhookQueue, err := webhookqueue.New(cf.WebhookQueueDir)
	if err != nil {
		log.Fatalf("incoming payment queue initialization failed: %v", err)
	}
	paymentsService := payments.NewPaymentsService(dbInstance)
	tgBotService, err := tg.NewTelegramService(cf.Token, paymentsService, cf.InvoicesChatID)
	if err != nil {
		log.Fatalf("error create tgbot %v", err)
	}

	maxBotService, err := max2.NewMaxService(cf.MaxToken, paymentsService, dbInstance)
	if err != nil {
		log.Fatalf("error create maxbot %v", err)
	}
	if err := maxBotService.RegisterBotCommands(); err != nil {
		log.Printf("register max bot commands: %v", err)
	}

	multiMessengers := multi.NewMultiMessenger([]usecase.SendMessanger{tgBotService, maxBotService})
	rnCardService := rncard.NewRnCardService(multiMessengers, dbInstance)
	banksService := banks.NewBankService(multiMessengers, dbInstance)
	banksHandler := handlers.NewBanksHandler(incomingWebhookQueue)
	telegramHandler := handlers.NewTelegramHandler(tgBotService)
	maxHandler := handlers.NewMaxHandler(maxBotService)
	miniAppHandler := handlers.NewMiniAppHandler(
		dbInstance,
		maxBotService,
		mailpkg.NewService(
			cf.SMTPHost,
			cf.SMTPPort,
			cf.SMTPUser,
			cf.SMTPPass,
			cf.SMTPFrom,
			cf.IMAPHost,
			cf.IMAPPort,
			cf.IMAPUser,
			cf.IMAPPass,
		),
	)

	router := transport.NewRouter(
		banksHandler,
		telegramHandler,
		maxHandler,
		miniAppHandler,
	)

	scheduler.SendDailyScheduler(rnCardService.FetchAndSendTransactions)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if os.Getenv("CODEX_ENABLED") == "true" {
		owner, e := strconv.ParseInt(os.Getenv("CODEX_MAX_OWNER_ID"), 10, 64)
		if e != nil || owner <= 0 || len(os.Getenv("MAX_WEBHOOK_SECRET")) < 32 || len(os.Getenv("CODEX_SERVICE_TOKEN")) < 32 || os.Getenv("CODEX_AGENT_URL") == "" {
			log.Fatal("invalid Codex configuration")
		}
		a := &codexagent.Service{DB: dbInstance.DB, URL: os.Getenv("CODEX_AGENT_URL"), Token: os.Getenv("CODEX_SERVICE_TOKEN"), Owner: owner, Send: maxBotService.SendDirectMessage}
		if e = a.Init(ctx); e != nil {
			log.Fatalf("codex queue: %v", e)
		}
		maxBotService.Agent = a
		go a.Run(ctx)
	}
	go incomingWebhookQueue.Run(ctx, func(event webhookqueue.Event) error {
		return banksService.ProcessWebhook(event.Source, event.Payload)
	})

	go (&accountant.Importer{
		DB: dbInstance,
		IMAP: mailpkg.IMAPConfig{
			Host:     cf.IMAPHost,
			Port:     cf.IMAPPort,
			User:     cf.IMAPUser,
			Pass:     cf.IMAPPass,
			Mailbox:  cf.IMAPMailbox,
			From:     cf.IMAPFrom,
			StartUID: cf.IMAPStartUID,
		},
		Max:           maxBotService,
		Telegram:      tgBotService,
		BackfillSince: cf.IMAPBackfillSince,
	}).Run(ctx)

	go sbis.NewMonitor(cf.SBISLogin, cf.SBISPassword, cf.SBISAccount, dbInstance, maxBotService.Bot).Run(ctx)

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
	cancel()

	log.Println("Shutdown signal received")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	err = tgBotService.SendMessageInGroupID(877804669, "Server stopped")
	if err != nil {
		log.Printf("error send server stopped message: %v", err)
	}

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}

	if err := dbInstance.DB.Close(); err != nil {
		log.Printf("db close error: %v", err)
	}

	log.Println("Server stopped")
}
