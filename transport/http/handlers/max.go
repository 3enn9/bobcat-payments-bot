package handlers

import (
	max2 "PaymentsBot/internal/max"
	"encoding/json"
	"github.com/max-messenger/max-bot-api-client-go/schemes"
	"io"
	"log"
	"net/http"
)

type MaxHandler struct {
	max *max2.MaxService
}

func NewMaxHandler(useCase *max2.MaxService) *MaxHandler {
	return &MaxHandler{max: useCase}
}

func (p *MaxHandler) MaxUpdates(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	body, _ := io.ReadAll(r.Body)
	log.Println(string(body))

	var update schemes.MessageCreatedUpdate
	if err := json.Unmarshal(body, &update); err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := p.max.Updates(&update); err != nil {
		log.Println(err)
	}

	w.WriteHeader(http.StatusOK)
}
