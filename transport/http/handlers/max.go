package handlers

import (
	max2 "PaymentsBot/internal/max"
	"encoding/json"
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

	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}

	update, err := max2.ParseWebhookUpdate(body)
	if err != nil {
		// fallback for legacy payloads without update_type
		var legacy struct {
			Message json.RawMessage `json:"message"`
		}
		if json.Unmarshal(body, &legacy) == nil && len(legacy.Message) > 0 {
			var msgUpd struct {
				UpdateType string `json:"update_type"`
			}
			_ = json.Unmarshal(body, &msgUpd)
			if msgUpd.UpdateType == "" {
				wrapped := append([]byte(`{"update_type":"message_created",`), body[1:]...)
				update, err = max2.ParseWebhookUpdate(wrapped)
			}
		}
		if err != nil {
			log.Printf("max webhook parse error: %v body=%s", err, string(body))
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	if update == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := p.max.Updates(update); err != nil {
		log.Println(err)
	}

	w.WriteHeader(http.StatusOK)
}
