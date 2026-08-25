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

	router.HandleFunc(
		"/api/miniapp/rogatka-requests",
		miniApp.ListRogatkaRequests,
	).Methods(http.MethodGet, http.MethodOptions)

	router.HandleFunc(
		"/api/miniapp/rogatka-requests/{id}/assign",
		miniApp.AssignRogatkaDriver,
	).Methods(http.MethodPost, http.MethodOptions)

	router.HandleFunc(
		"/api/miniapp/rogatka-requests/{id}/complete",
		miniApp.CompleteRogatkaRequest,
	).Methods(http.MethodPost, http.MethodOptions)

	router.HandleFunc(
		"/api/miniapp/rogatka-requests/{id}",
		miniApp.DeleteRogatkaRequest,
	).Methods(http.MethodDelete, http.MethodOptions)

	return router
}
