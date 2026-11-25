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
	"PaymentsBot/internal/db"
	"PaymentsBot/internal/invoice"
	max2 "PaymentsBot/internal/max"

	"github.com/gorilla/mux"
)

func (h *MiniAppHandler) SuggestBunkerHaulSurnames(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if len([]rune(query)) > 128 {
		http.Error(w, `{"success":false,"error":"Фамилия слишком длинная"}`, http.StatusBadRequest)
		return
	}

	surnames, err := h.db.SuggestBunkerHaulSurnames(query)
	if err != nil {
		log.Printf("suggest bunker haul surnames query=%q: %v", query, err)
		http.Error(w, `{"success":false,"error":"Ошибка загрузки фамилий"}`, http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"surnames": surnames,
	})
}

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

func (h *MiniAppHandler) GetBunkerHaulReportComment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	surname := strings.TrimSpace(r.URL.Query().Get("surname"))
	dateText := strings.TrimSpace(r.URL.Query().Get("date"))
	date, err := time.ParseInLocation("2006-01-02", dateText, clock.Now().Location())
	if surname == "" || len([]rune(surname)) > 128 || err != nil || date.Format("2006-01-02") != dateText {
		http.Error(w, `{"success":false,"error":"Укажите корректные фамилию и дату"}`, http.StatusBadRequest)
		return
	}

	comments, err := h.db.GetBunkerHaulReportComment(surname, date)
	if err != nil {
		log.Printf("load bunker haul report comment surname=%q date=%s: %v", surname, dateText, err)
		http.Error(w, `{"success":false,"error":"Не удалось загрузить комментарий"}`, http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "comments": comments})
}

func (h *MiniAppHandler) SaveBunkerHaulReportComment(w http.ResponseWriter, r *http.Request) {
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
	date, err := time.ParseInLocation("2006-01-02", input.Date, clock.Now().Location())
	if input.Surname == "" || len([]rune(input.Surname)) > 128 || err != nil || date.Format("2006-01-02") != input.Date {
		http.Error(w, `{"success":false,"error":"Укажите корректные фамилию и дату"}`, http.StatusBadRequest)
		return
	}
	if len([]rune(input.Comments)) > 2000 {
		http.Error(w, `{"success":false,"error":"Комментарий слишком длинный"}`, http.StatusBadRequest)
		return
	}
	if err := h.db.SaveBunkerHaulReportComment(input.Surname, date, input.Comments); err != nil {
		log.Printf("save bunker haul report comment surname=%q date=%s: %v", input.Surname, input.Date, err)
		http.Error(w, `{"success":false,"error":"Не удалось сохранить комментарий"}`, http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (h *MiniAppHandler) SendBunkerHaulReport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	var input struct {
		Surname      string  `json:"surname"`
		Date         string  `json:"date"`
		Comments     string  `json:"comments"`
		CashEntryIDs []int64 `json:"cashEntryIds"`
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
	if len(input.CashEntryIDs) > 50 {
		http.Error(w, `{"success":false,"error":"Слишком много операций кассы"}`, http.StatusBadRequest)
		return
	}
	seenCashEntries := make(map[int64]struct{}, len(input.CashEntryIDs))
	for _, id := range input.CashEntryIDs {
		if id <= 0 {
			http.Error(w, `{"success":false,"error":"Некорректная операция кассы"}`, http.StatusBadRequest)
			return
		}
		if _, exists := seenCashEntries[id]; exists {
			http.Error(w, `{"success":false,"error":"Операции кассы не должны повторяться"}`, http.StatusBadRequest)
			return
		}
		seenCashEntries[id] = struct{}{}
	}
	date, err := time.ParseInLocation("2006-01-02", input.Date, clock.Now().Location())
	if err != nil || date.Format("2006-01-02") != input.Date {
		http.Error(w, `{"success":false,"error":"Укажите корректную дату"}`, http.StatusBadRequest)
		return
	}
	if err := h.db.SaveBunkerHaulReportComment(input.Surname, date, input.Comments); err != nil {
		log.Printf("save report comment before send surname=%q date=%s: %v", input.Surname, input.Date, err)
		http.Error(w, `{"success":false,"error":"Не удалось сохранить комментарий отчёта"}`, http.StatusInternalServerError)
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
	for _, item := range items {
		if err := h.db.SaveBunkerHaulAnnotations(item.ID, input.Surname, date, item.Annotations); err != nil {
			log.Printf("sync bunker haul cash entries id=%d surname=%q date=%s: %v", item.ID, input.Surname, input.Date, err)
			http.Error(w, `{"success":false,"error":"Не удалось сохранить операции в кассу"}`, http.StatusInternalServerError)
			return
		}
	}
	cashEntries, err := h.db.ListWorkerCashEntriesByIDs(input.Surname, date, input.CashEntryIDs)
	if err != nil {
		log.Printf("load report cash entries surname=%q date=%s: %v", input.Surname, input.Date, err)
		http.Error(w, `{"success":false,"error":"Не удалось загрузить операции кассы"}`, http.StatusInternalServerError)
		return
	}
	if len(cashEntries) != len(input.CashEntryIDs) {
		http.Error(w, `{"success":false,"error":"Операция кассы не найдена для выбранной фамилии и даты"}`, http.StatusBadRequest)
		return
	}

	balance, err := h.db.GetWorkerCashBalance(input.Surname)
	if err != nil {
		log.Printf("load worker cash balance surname=%q: %v", input.Surname, err)
		http.Error(w, `{"success":false,"error":"Не удалось загрузить кассу работника"}`, http.StatusInternalServerError)
		return
	}
	reportMessages := make([]invoice.BunkerHaulReportMessage, 0, len(items))
	for _, item := range items {
		reportMessage := invoice.BunkerHaulReportMessage{
			SentAt:      item.SentAt,
			Text:        item.Text,
			Annotations: make([]invoice.BunkerHaulReportAnnotation, 0, len(item.Annotations)),
		}
		for _, annotation := range item.Annotations {
			reportMessage.Annotations = append(reportMessage.Annotations, invoice.BunkerHaulReportAnnotation{
				Kind:   annotation.Kind,
				Amount: annotation.Amount,
			})
		}
		reportMessages = append(reportMessages, reportMessage)
	}
	reportCashOperations := make([]invoice.BunkerHaulReportCashOperation, 0, len(cashEntries))
	for _, entry := range cashEntries {
		reportCashOperations = append(reportCashOperations, invoice.BunkerHaulReportCashOperation{
			EntryType:   entry.EntryType,
			Amount:      entry.Amount,
			Description: entry.Description,
		})
	}
	pdf, err := invoice.GenerateBunkerHaulReportPDF(input.Surname, date, reportMessages, reportCashOperations, input.Comments, balance)
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
		Surname     string                    `json:"surname"`
		Date        string                    `json:"date"`
		Annotations []db.BunkerHaulAnnotation `json:"annotations"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"success":false,"error":"Некорректные данные"}`, http.StatusBadRequest)
		return
	}
	input.Surname = strings.TrimSpace(input.Surname)
	if input.Surname == "" || len([]rune(input.Surname)) > 128 {
		http.Error(w, `{"success":false,"error":"Укажите корректную фамилию"}`, http.StatusBadRequest)
		return
	}
	date, err := time.ParseInLocation("2006-01-02", input.Date, clock.Now().Location())
	if err != nil || date.Format("2006-01-02") != input.Date {
		http.Error(w, `{"success":false,"error":"Укажите корректную дату"}`, http.StatusBadRequest)
		return
	}
	if len(input.Annotations) > 5 {
		http.Error(w, `{"success":false,"error":"Можно выбрать не более пяти отметок"}`, http.StatusBadRequest)
		return
	}
	seen := make(map[string]struct{}, len(input.Annotations))
	for _, annotation := range input.Annotations {
		if annotation.Kind != "vector" && annotation.Kind != "dubki" && annotation.Kind != "cash" && annotation.Kind != "base" && annotation.Kind != "rsm" {
			http.Error(w, `{"success":false,"error":"Выберите корректные отметки"}`, http.StatusBadRequest)
			return
		}
		if annotation.Amount <= 0 || annotation.Amount > 999999999.999 || math.IsNaN(annotation.Amount) || math.IsInf(annotation.Amount, 0) {
			http.Error(w, `{"success":false,"error":"Значение отметки должно быть больше нуля"}`, http.StatusBadRequest)
			return
		}
		if (annotation.Kind == "base" || annotation.Kind == "rsm") && annotation.Amount != 1 {
			http.Error(w, `{"success":false,"error":"Отметки База и РСМ принимают только значение да"}`, http.StatusBadRequest)
			return
		}
		if _, ok := seen[annotation.Kind]; ok {
			http.Error(w, `{"success":false,"error":"Отметки не должны повторяться"}`, http.StatusBadRequest)
			return
		}
		seen[annotation.Kind] = struct{}{}
	}

	if err := h.db.SaveBunkerHaulAnnotations(haulID, input.Surname, date, input.Annotations); err != nil {
		log.Printf("save bunker haul annotation id=%d: %v", haulID, err)
		http.Error(w, `{"success":false,"error":"Не удалось сохранить отметку"}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
}
