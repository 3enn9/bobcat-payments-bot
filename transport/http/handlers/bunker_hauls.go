package handlers

import (
	"PaymentsBot/internal/clock"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
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

func (h *MiniAppHandler) SaveBunkerHaulAnnotation(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	haulID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil || haulID <= 0 {
		http.Error(w, `{"success":false,"error":"Некорректное сообщение"}`, http.StatusBadRequest)
		return
	}

	var input struct {
		Kind  string  `json:"kind"`
		Value float64 `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"success":false,"error":"Некорректные данные"}`, http.StatusBadRequest)
		return
	}
	if input.Kind != "vector" && input.Kind != "dubki" && input.Kind != "cash" {
		http.Error(w, `{"success":false,"error":"Выберите способ"}`, http.StatusBadRequest)
		return
	}
	if input.Value <= 0 || input.Value > 999999999.999 || math.IsNaN(input.Value) || math.IsInf(input.Value, 0) {
		http.Error(w, `{"success":false,"error":"Введите корректное значение"}`, http.StatusBadRequest)
		return
	}

	if err := h.db.SaveBunkerHaulAnnotation(haulID, input.Kind, input.Value); err != nil {
		log.Printf("save bunker haul annotation id=%d: %v", haulID, err)
		http.Error(w, `{"success":false,"error":"Не удалось сохранить отметку"}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
}
