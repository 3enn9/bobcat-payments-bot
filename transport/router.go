package transport

import (
	"PaymentsBot/transport/http/handlers"
	"net/http"

	"github.com/gorilla/mux"
)

func NewRouter(
	banks *handlers.BanksHandler,
	telegram *handlers.TelegramHandler,
	max *handlers.MaxHandler,
	miniApp *handlers.MiniAppHandler,
) *mux.Router {
	router := mux.NewRouter().StrictSlash(true)

	router.HandleFunc(
		"/telegram/webhook",
		telegram.TelegramUpdates,
	)

	router.HandleFunc(
		"/max/webhook",
		max.MaxUpdates,
	)

	router.HandleFunc(
		"/webhook",
		banks.TochkaBankHandler,
	)

	router.HandleFunc(
		"/modulbank",
		banks.ModuleBankHandler,
	)

	router.HandleFunc(
		"/tbank",
		banks.TBankHandler,
	)

	router.HandleFunc(
		"/api/miniapp/requests",
		miniApp.CreateRequest,
	).Methods(http.MethodPost, http.MethodOptions)

	return router
}
