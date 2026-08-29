package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"PaymentsBot/internal/clock"
	"PaymentsBot/internal/db"
)

type createDaysOffRequest struct {
	WorkerName string `json:"workerName"`
	DateFrom   string `json:"dateFrom"`
	DateTo     string `json:"dateTo"`
	Comment    string `json:"comment"`
}

func (h *MiniAppHandler) CreateDaysOff(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	var input createDaysOffRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"success":false,"error":"Некорректный JSON"}`, http.StatusBadRequest)
		return
	}

	input.WorkerName = strings.TrimSpace(input.WorkerName)
	input.DateFrom = strings.TrimSpace(input.DateFrom)
	input.DateTo = strings.TrimSpace(input.DateTo)
	input.Comment = strings.TrimSpace(input.Comment)

	if input.WorkerName == "" {
		http.Error(w, `{"success":false,"error":"Укажите фамилию"}`, http.StatusBadRequest)
		return
	}
	if input.DateFrom == "" || input.DateTo == "" {
		http.Error(w, `{"success":false,"error":"Укажите даты"}`, http.StatusBadRequest)
		return
	}
	if _, err := time.Parse("2006-01-02", input.DateFrom); err != nil {
		http.Error(w, `{"success":false,"error":"Некорректная дата начала"}`, http.StatusBadRequest)
		return
	}
	if _, err := time.Parse("2006-01-02", input.DateTo); err != nil {
		http.Error(w, `{"success":false,"error":"Некорректная дата окончания"}`, http.StatusBadRequest)
		return
	}
	if input.DateTo < input.DateFrom {
		http.Error(w, `{"success":false,"error":"Дата окончания раньше начала"}`, http.StatusBadRequest)
		return
	}

	today := clock.Today()
	if input.DateFrom < today {
		http.Error(w, `{"success":false,"error":"Нельзя выбрать прошедшие даты"}`, http.StatusBadRequest)
		return
	}

	id, err := h.db.CreateWorkerDaysOff(input.WorkerName, input.DateFrom, input.DateTo, input.Comment)
	if err != nil {
		log.Printf("create days off error: %v", err)
		http.Error(w, `{"success":false,"error":"Ошибка сохранения"}`, http.StatusInternalServerError)
		return
	}

	item := db.WorkerDaysOff{
		ID:         id,
		WorkerName: input.WorkerName,
		DateFrom:   input.DateFrom,
		DateTo:     input.DateTo,
		Comment:    input.Comment,
		Status:     "pending",
	}

	messageID, err := h.max.SendDaysOffApproval(item, id)
	if err != nil {
		log.Printf("send days off to MAX error id=%d: %v", id, err)
		http.Error(w, `{"success":false,"error":"Заявка сохранена, но не удалось отправить в группу"}`, http.StatusInternalServerError)
		return
	}

	if err := h.db.SetWorkerDaysOffMessage(id, messageID, -78302034737888); err != nil {
		log.Printf("save days off message id error: %v", err)
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"id":      id,
	})
}

func (h *MiniAppHandler) ListDaysOff(w http.ResponseWriter, r *http.Request) {
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

	items, err := h.db.ListWorkerDaysOff(workerName, clock.Today(), 30)
	if err != nil {
		log.Printf("list days off error: %v", err)
		http.Error(w, `{"success":false,"error":"Ошибка загрузки"}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"items":   items,
	})
}

func (h *MiniAppHandler) ListUpcomingDaysOff(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	items, err := h.db.ListUpcomingApprovedDaysOff(clock.Today(), 50)
	if err != nil {
		log.Printf("list upcoming days off error: %v", err)
		http.Error(w, `{"success":false,"error":"Ошибка загрузки"}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"items":   items,
	})
}
