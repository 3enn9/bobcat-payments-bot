package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"PaymentsBot/internal/db"
)

type createGarageWorkRequest struct {
	WorkerName  string `json:"workerName"`
	WorkDate    string `json:"workDate"`
	TimeFrom    string `json:"timeFrom"`
	TimeTo      string `json:"timeTo"`
	Description string `json:"description"`
}

func (h *MiniAppHandler) CreateGarageWork(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	var input createGarageWorkRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"success":false,"error":"Некорректный JSON"}`, http.StatusBadRequest)
		return
	}

	input.WorkerName = strings.TrimSpace(input.WorkerName)
	input.WorkDate = strings.TrimSpace(input.WorkDate)
	input.TimeFrom = strings.TrimSpace(input.TimeFrom)
	input.TimeTo = strings.TrimSpace(input.TimeTo)
	input.Description = strings.TrimSpace(input.Description)

	if input.WorkerName == "" {
		http.Error(w, `{"success":false,"error":"Укажите фамилию"}`, http.StatusBadRequest)
		return
	}
	if input.WorkDate == "" {
		http.Error(w, `{"success":false,"error":"Укажите дату"}`, http.StatusBadRequest)
		return
	}
	if _, err := time.Parse("2006-01-02", input.WorkDate); err != nil {
		http.Error(w, `{"success":false,"error":"Некорректная дата"}`, http.StatusBadRequest)
		return
	}
	if input.TimeFrom == "" || input.TimeTo == "" {
		http.Error(w, `{"success":false,"error":"Укажите время начала и окончания"}`, http.StatusBadRequest)
		return
	}
	if !isValidTimeHHMM(input.TimeFrom) || !isValidTimeHHMM(input.TimeTo) {
		http.Error(w, `{"success":false,"error":"Некорректное время"}`, http.StatusBadRequest)
		return
	}
	if input.TimeTo <= input.TimeFrom {
		http.Error(w, `{"success":false,"error":"Время окончания должно быть позже начала"}`, http.StatusBadRequest)
		return
	}
	if input.Description == "" {
		http.Error(w, `{"success":false,"error":"Укажите описание работы"}`, http.StatusBadRequest)
		return
	}

	timeFrom := normalizeTimeHHMM(input.TimeFrom)
	timeTo := normalizeTimeHHMM(input.TimeTo)
	workedMinutes, err := calcWorkedMinutes(input.TimeFrom, input.TimeTo)
	if err != nil || workedMinutes <= 0 {
		http.Error(w, `{"success":false,"error":"Не удалось посчитать отработанное время"}`, http.StatusBadRequest)
		return
	}

	id, err := h.db.CreateGarageWorkLog(
		input.WorkerName,
		input.WorkDate,
		timeFrom,
		timeTo,
		workedMinutes,
		input.Description,
	)
	if err != nil {
		log.Printf("create garage work error: %v", err)
		http.Error(w, `{"success":false,"error":"Ошибка сохранения"}`, http.StatusInternalServerError)
		return
	}

	item := db.GarageWorkLog{
		ID:            id,
		WorkerName:    input.WorkerName,
		WorkDate:      input.WorkDate,
		TimeFrom:      timeFrom,
		TimeTo:        timeTo,
		WorkedMinutes: workedMinutes,
		Description:   input.Description,
	}
	if err := h.max.SendGarageWorkReport(item); err != nil {
		log.Printf("send garage work to MAX error id=%d: %v", id, err)
		http.Error(w, `{"success":false,"error":"Запись сохранена, но не удалось отправить в группу"}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"id":      id,
	})
}

func isValidTimeHHMM(value string) bool {
	_, err := time.Parse("15:04", value)
	return err == nil
}

func normalizeTimeHHMM(value string) string {
	if t, err := time.Parse("15:04", value); err == nil {
		return t.Format("15:04:05")
	}
	return value
}

func calcWorkedMinutes(timeFrom, timeTo string) (int, error) {
	from, err := parseClockTime(timeFrom)
	if err != nil {
		return 0, err
	}
	to, err := parseClockTime(timeTo)
	if err != nil {
		return 0, err
	}
	minutes := int(to.Sub(from).Minutes())
	if minutes <= 0 {
		return 0, err
	}
	return minutes, nil
}

func parseClockTime(value string) (time.Time, error) {
	if t, err := time.Parse("15:04", value); err == nil {
		return t, nil
	}
	return time.Parse("15:04:05", value)
}
