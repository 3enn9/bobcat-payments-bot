package main

import (
	banks "PaymentsBot/internal/banks"
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/webhook", banks.TochkaBankHandler)
	http.HandleFunc("/modulbank", banks.ModuleBankHandler)
	http.HandleFunc("/tbank", banks.TBankHandler)
	fmt.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
