package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type IncomingPayments struct {
	ID        string  `json:"id"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	From      string  `json:"from"`
	To        string  `json:"to"`
	Timestamp string  `json:"timestamp"`
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusOK)
		return
	}
	var payment IncomingPayments

	if err := json.NewDecoder(r.Body).Decode(&payment); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	fmt.Printf("Получен платеж: %+v\n", payment)
	w.WriteHeader(http.StatusOK)
}

func main() {
	http.HandleFunc("/webhook", webhookHandler)
	fmt.Println("Сервер запущен на порту :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
