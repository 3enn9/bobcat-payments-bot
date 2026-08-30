package handlers

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"strings"
	"time"
)

type createCashEntryRequest struct {
	WorkerName  string  `json:"workerName"`
	EntryType   string  `json:"entryType"`
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
	EntryDate   string  `json:"entryDate"`
}

func (h *MiniAppHandler) ListWorkerCash(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	workerName := strings.TrimSpace(r.URL.Query().Get("worker"))
	if workerName == "" {
		http.Error(w, `{"success":false,"error":"Укажите фамилию"}`, http.StatusBadRequest)
		return
	}

	entries, err := h.db.ListWorkerCashEntries(workerName)
	if err != nil {
		log.Printf("list worker cash error: %v", err)
		http.Error(w, `{"success":false,"error":"Ошибка загрузки"}`, http.StatusInternalServerError)
		return
	}

	balance, err := h.db.GetWorkerCashBalance(workerName)
	if err != nil {
		log.Printf("worker cash balance error: %v", err)
		http.Error(w, `{"success":false,"error":"Ошибка загрузки"}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"balance": balance,
		"entries": entries,
	})
}

func (h *MiniAppHandler) CreateWorkerCashEntry(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	var input createCashEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"success":false,"error":"Некорректный JSON"}`, http.StatusBadRequest)
		return
	}

	input.WorkerName = strings.TrimSpace(input.WorkerName)
	input.EntryType = strings.TrimSpace(input.EntryType)
	input.Description = strings.TrimSpace(input.Description)
	input.EntryDate = strings.TrimSpace(input.EntryDate)

	if input.WorkerName == "" {
		http.Error(w, `{"success":false,"error":"Укажите фамилию"}`, http.StatusBadRequest)
		return
	}
	if len([]rune(input.WorkerName)) > 100 {
		http.Error(w, `{"success":false,"error":"Фамилия слишком длинная"}`, http.StatusBadRequest)
		return
	}
	switch input.EntryType {
	case "income", "expense":
	default:
		http.Error(w, `{"success":false,"error":"Укажите тип: приход или расход"}`, http.StatusBadRequest)
		return
	}
	if input.Amount <= 0 || math.IsNaN(input.Amount) || math.IsInf(input.Amount, 0) {
		http.Error(w, `{"success":false,"error":"Сумма должна быть больше нуля"}`, http.StatusBadRequest)
		return
	}
	if input.Amount > 999999999.99 {
		http.Error(w, `{"success":false,"error":"Слишком большая сумма"}`, http.StatusBadRequest)
		return
	}
	amount := math.Round(input.Amount*100) / 100

	if input.EntryDate == "" {
		input.EntryDate = time.Now().Format("2006-01-02")
	}
	if _, err := time.Parse("2006-01-02", input.EntryDate); err != nil {
		http.Error(w, `{"success":false,"error":"Некорректная дата"}`, http.StatusBadRequest)
		return
	}
	if len([]rune(input.Description)) > 500 {
		http.Error(w, `{"success":false,"error":"Описание слишком длинное"}`, http.StatusBadRequest)
		return
	}

	id, err := h.db.CreateWorkerCashEntry(
		input.WorkerName,
		input.EntryType,
		amount,
		input.Description,
		input.EntryDate,
	)
	if err != nil {
		log.Printf("create worker cash entry error: %v", err)
		http.Error(w, `{"success":false,"error":"Ошибка сохранения"}`, http.StatusInternalServerError)
		return
	}

	balance, err := h.db.GetWorkerCashBalance(input.WorkerName)
	if err != nil {
		log.Printf("worker cash balance after create error: %v", err)
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"id":      id,
		"balance": balance,
	})
}
