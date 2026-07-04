package transport

import (
	"PaymentsBot/transport/http/handlers"
	"github.com/gorilla/mux"
)

func NewRouter(banks *handlers.BanksHandler, telegram *handlers.TelegramHandler) *mux.Router {
	router := mux.NewRouter().StrictSlash(true)

	router.HandleFunc("/telegram/webhook", telegram.TelegramUpdates)
	router.HandleFunc("/webhook", banks.TochkaBankHandler)
	router.HandleFunc("/modulbank", banks.ModuleBankHandler)
	router.HandleFunc("/tbank", banks.TBankHandler)

	return router
}
