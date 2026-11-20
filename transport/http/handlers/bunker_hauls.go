package handlers

import (
	"PaymentsBot/internal/clock"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

func (h *MiniAppHandler) ListBunkerHaulMessages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	surname := strings.TrimSpace(r.URL.Query().Get("surname"))
	if surname == "" {
		http.Error(w, `{"success":false,"error":"Укажите фамилию работника"}`, http.StatusBadRequest)
		return
	}
	if len([]rune(surname)) > 128 {
		http.Error(w, `{"success":false,"error":"Фамилия слишком длинная"}`, http.StatusBadRequest)
		return
	}

	dateText := strings.TrimSpace(r.URL.Query().Get("date"))
	date, err := time.ParseInLocation("2006-01-02", dateText, clock.Now().Location())
	if err != nil || date.Format("2006-01-02") != dateText {
		http.Error(w, `{"success":false,"error":"Укажите дату в формате YYYY-MM-DD"}`, http.StatusBadRequest)
		return
	}

	items, err := h.db.ListBunkerHaulMessagesBySurname(surname, date)
	if err != nil {
		log.Printf("list bunker haul messages surname=%q date=%s: %v", surname, dateText, err)
		http.Error(w, `{"success":false,"error":"Ошибка загрузки сообщений"}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"items":   items,
	})
}
