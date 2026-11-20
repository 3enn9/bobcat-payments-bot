package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"PaymentsBot/internal/clock"
	"PaymentsBot/internal/invoice"
	max2 "PaymentsBot/internal/max"

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

func (h *MiniAppHandler) SendBunkerHaulReport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	var input struct {
		Surname  string `json:"surname"`
		Date     string `json:"date"`
		Comments string `json:"comments"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"success":false,"error":"Некорректные данные"}`, http.StatusBadRequest)
		return
	}
	input.Surname = strings.TrimSpace(input.Surname)
	input.Comments = strings.TrimSpace(input.Comments)
	if input.Surname == "" || len([]rune(input.Surname)) > 128 {
		http.Error(w, `{"success":false,"error":"Укажите корректную фамилию"}`, http.StatusBadRequest)
		return
	}
	if len([]rune(input.Comments)) > 2000 {
		http.Error(w, `{"success":false,"error":"Комментарий слишком длинный"}`, http.StatusBadRequest)
		return
	}
	date, err := time.ParseInLocation("2006-01-02", input.Date, clock.Now().Location())
	if err != nil || date.Format("2006-01-02") != input.Date {
		http.Error(w, `{"success":false,"error":"Укажите корректную дату"}`, http.StatusBadRequest)
		return
	}

	items, err := h.db.ListBunkerHaulMessagesBySurname(input.Surname, date)
	if err != nil {
		log.Printf("load bunker haul report surname=%q date=%s: %v", input.Surname, input.Date, err)
		http.Error(w, `{"success":false,"error":"Не удалось загрузить сообщения"}`, http.StatusInternalServerError)
		return
	}
	if len(items) == 0 {
		http.Error(w, `{"success":false,"error":"За выбранную дату нет сообщений"}`, http.StatusBadRequest)
		return
	}

	reportMessages := make([]invoice.BunkerHaulReportMessage, 0, len(items))
	for _, item := range items {
		reportMessages = append(reportMessages, invoice.BunkerHaulReportMessage{
			SentAt:         item.SentAt,
			Text:           item.Text,
			AnnotationKind: item.AnnotationKind,
			Amount:         item.AnnotationAmount,
		})
	}
	pdf, err := invoice.GenerateBunkerHaulReportPDF(input.Surname, date, reportMessages, input.Comments)
	if err != nil {
		log.Printf("generate bunker haul report surname=%q date=%s: %v", input.Surname, input.Date, err)
		http.Error(w, `{"success":false,"error":"Не удалось сформировать отчёт"}`, http.StatusInternalServerError)
		return
	}
	pages, err := invoice.PDFToImages(pdf, 150)
	if err != nil {
		log.Printf("render bunker haul report surname=%q date=%s: %v", input.Surname, input.Date, err)
		http.Error(w, `{"success":false,"error":"Не удалось подготовить фото отчёта"}`, http.StatusInternalServerError)
		return
	}
	photos := make([]max2.PhotoUpload, 0, len(pages))
	for i, page := range pages {
		photos = append(photos, max2.PhotoUpload{
			Name:   fmt.Sprintf("bunker-hauls-%d.png", i+1),
			Reader: bytes.NewReader(page),
		})
	}
	caption := fmt.Sprintf("%s — %s", input.Surname, date.Format("02.01.2006"))
	if err := h.max.SendBunkerHaulReport(caption, photos); err != nil {
		log.Printf("send bunker haul report surname=%q date=%s: %v", input.Surname, input.Date, err)
		http.Error(w, `{"success":false,"error":"Не удалось отправить отчёт в MAX"}`, http.StatusBadGateway)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
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
