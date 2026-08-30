package handlers

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type cashEntryRequest struct {
	WorkerName  string  `json:"workerName"`
	EntryType   string  `json:"entryType"`
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
	EntryDate   string  `json:"entryDate"`
}

func parseCashEntryLimit(raw string) int {
	if raw == "" {
		return 10
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit <= 0 {
		return 10
	}
	if limit > 50 {
		return 50
	}
	return limit
}

func normalizeCashEntryInput(input *cashEntryRequest, defaultDate bool) (amount float64, errMsg string) {
	input.WorkerName = strings.TrimSpace(input.WorkerName)
	input.EntryType = strings.TrimSpace(input.EntryType)
	input.Description = strings.TrimSpace(input.Description)
	input.EntryDate = strings.TrimSpace(input.EntryDate)

	if input.WorkerName == "" {
		return 0, "Укажите фамилию"
	}
	if len([]rune(input.WorkerName)) > 100 {
		return 0, "Фамилия слишком длинная"
	}
	switch input.EntryType {
	case "income", "expense":
	default:
		return 0, "Укажите тип: приход или расход"
	}
	if input.Amount <= 0 || math.IsNaN(input.Amount) || math.IsInf(input.Amount, 0) {
		return 0, "Сумма должна быть больше нуля"
	}
	if input.Amount > 999999999.99 {
		return 0, "Слишком большая сумма"
	}
	if input.EntryDate == "" {
		if defaultDate {
			input.EntryDate = time.Now().Format("2006-01-02")
		} else {
			return 0, "Укажите дату"
		}
	}
	if _, err := time.Parse("2006-01-02", input.EntryDate); err != nil {
		return 0, "Некорректная дата"
	}
	if len([]rune(input.Description)) > 500 {
		return 0, "Описание слишком длинное"
	}

	return math.Round(input.Amount*100) / 100, ""
}

func (h *MiniAppHandler) ListCashWorkers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	h.listCashWorkersJSON(w)
}

func (h *MiniAppHandler) listCashWorkersJSON(w http.ResponseWriter) {
	names, err := h.db.ListCashWorkerNames()
	if err != nil {
		log.Printf("list cash workers error: %v", err)
		http.Error(w, `{"success":false,"error":"Ошибка загрузки"}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"workers": names,
	})
}

func (h *MiniAppHandler) ListWorkerCash(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.URL.Query().Get("workers") == "1" {
		h.listCashWorkersJSON(w)
		return
	}

	workerName := strings.TrimSpace(r.URL.Query().Get("worker"))
	if workerName == "" {
		http.Error(w, `{"success":false,"error":"Укажите фамилию"}`, http.StatusBadRequest)
		return
	}

	limit := parseCashEntryLimit(r.URL.Query().Get("limit"))

	entries, err := h.db.ListWorkerCashEntries(workerName, limit)
	if err != nil {
		log.Printf("list worker cash error: %v", err)
		http.Error(w, `{"success":false,"error":"Ошибка загрузки"}`, http.StatusInternalServerError)
		return
	}

	total, err := h.db.CountWorkerCashEntries(workerName)
	if err != nil {
		log.Printf("count worker cash error: %v", err)
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
		"total":   total,
		"limit":   limit,
	})
}

func (h *MiniAppHandler) CreateWorkerCashEntry(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	var input cashEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"success":false,"error":"Некорректный JSON"}`, http.StatusBadRequest)
		return
	}

	amount, errMsg := normalizeCashEntryInput(&input, true)
	if errMsg != "" {
		http.Error(w, `{"success":false,"error":"`+errMsg+`"}`, http.StatusBadRequest)
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

func (h *MiniAppHandler) UpdateWorkerCashEntry(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, `{"success":false,"error":"Некорректный ID"}`, http.StatusBadRequest)
		return
	}

	var input cashEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"success":false,"error":"Некорректный JSON"}`, http.StatusBadRequest)
		return
	}

	amount, errMsg := normalizeCashEntryInput(&input, false)
	if errMsg != "" {
		http.Error(w, `{"success":false,"error":"`+errMsg+`"}`, http.StatusBadRequest)
		return
	}

	ok, err := h.db.UpdateWorkerCashEntry(
		id,
		input.WorkerName,
		input.EntryType,
		amount,
		input.Description,
		input.EntryDate,
	)
	if err != nil {
		log.Printf("update worker cash entry error id=%d: %v", id, err)
		http.Error(w, `{"success":false,"error":"Ошибка сохранения"}`, http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, `{"success":false,"error":"Запись не найдена"}`, http.StatusNotFound)
		return
	}

	balance, err := h.db.GetWorkerCashBalance(input.WorkerName)
	if err != nil {
		log.Printf("worker cash balance after update error: %v", err)
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"balance": balance,
	})
}
