package banks

import (
	"log"
	"net/http"
)

func TBankHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("ТБанк, пришел POST запрос:\n%v", r.Body)

	w.WriteHeader(http.StatusOK)
}
