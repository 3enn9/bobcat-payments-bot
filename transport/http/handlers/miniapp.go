package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"PaymentsBot/internal/db"
	max2 "PaymentsBot/internal/max"
	mailpkg "PaymentsBot/internal/mail"

	"github.com/gorilla/mux"
)

type MiniAppHandler struct {
	db   *db.Database
	max  *max2.MaxService
	mail *mailpkg.Service
}

func NewMiniAppHandler(database *db.Database, maxService *max2.MaxService, mailService *mailpkg.Service) *MiniAppHandler {
	return &MiniAppHandler{
		db:   database,
		max:  maxService,
		mail: mailService,
	}
}

type createMiniAppRequest struct {
	MaxUserID   string `json:"maxUserId"`
	MaxUsername string `json:"maxUsername"`
	Name        string `json:"name"`
	Contact     string `json:"contact"`
	Message     string `json:"message"`
}

func (h *MiniAppHandler) CreateRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var input createMiniAppRequest

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"success":false,"error":"Некорректный JSON"}`, http.StatusBadRequest)
		return
	}

	input.Name = strings.TrimSpace(input.Name)
	input.Contact = strings.TrimSpace(input.Contact)
	input.Message = strings.TrimSpace(input.Message)

	if input.Name == "" || input.Contact == "" || input.Message == "" {
		http.Error(
			w,
			`{"success":false,"error":"Заполните все поля"}`,
			http.StatusBadRequest,
		)
		return
	}

	if len([]rune(input.Name)) > 100 {
		http.Error(
			w,
			`{"success":false,"error":"Имя слишком длинное"}`,
			http.StatusBadRequest,
		)
		return
	}

	if len([]rune(input.Contact)) > 150 {
		http.Error(
			w,
			`{"success":false,"error":"Контакт слишком длинный"}`,
			http.StatusBadRequest,
		)
		return
	}

	if len([]rune(input.Message)) > 2000 {
		http.Error(
			w,
			`{"success":false,"error":"Сообщение слишком длинное"}`,
			http.StatusBadRequest,
		)
		return
	}

	id, err := h.db.CreateMiniAppRequest(
		input.MaxUserID,
		input.MaxUsername,
		input.Name,
		input.Contact,
		input.Message,
	)

	if err != nil {
		http.Error(
			w,
			`{"success":false,"error":"Ошибка сохранения заявки"}`,
			http.StatusInternalServerError,
		)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"id":      id,
	})
}

func (h *MiniAppHandler) ListRogatkaRequests(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	status := strings.TrimSpace(r.URL.Query().Get("status"))
	driverName := strings.TrimSpace(r.URL.Query().Get("driver"))
	assigned := false

	switch {
	case driverName != "":
		assigned = true
	case status == "" || status == "active":
		assigned = false
	case status == "assigned":
		assigned = true
	default:
		http.Error(w, `{"success":false,"error":"Некорректный статус"}`, http.StatusBadRequest)
		return
	}

	requests, err := h.db.ListRogatkaRequests(assigned, driverName)
	if err != nil {
		http.Error(
			w,
			`{"success":false,"error":"Ошибка загрузки заявок"}`,
			http.StatusInternalServerError,
		)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"requests": requests,
	})
}

type assignRogatkaDriverRequest struct {
	DriverName string `json:"driverName"`
}

func (h *MiniAppHandler) AssignRogatkaDriver(w http.ResponseWriter, r *http.Request) {
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

	var input assignRogatkaDriverRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"success":false,"error":"Некорректный JSON"}`, http.StatusBadRequest)
		return
	}

	input.DriverName = strings.TrimSpace(input.DriverName)
	if input.DriverName == "" {
		http.Error(w, `{"success":false,"error":"Укажите фамилию водителя"}`, http.StatusBadRequest)
		return
	}
	if len([]rune(input.DriverName)) > 100 {
		http.Error(w, `{"success":false,"error":"Фамилия слишком длинная"}`, http.StatusBadRequest)
		return
	}

	message, ok, err := h.db.AssignRogatkaDriver(id, input.DriverName)
	if err != nil {
		http.Error(w, `{"success":false,"error":"Ошибка назначения водителя"}`, http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, `{"success":false,"error":"Заявка не найдена или уже назначена"}`, http.StatusNotFound)
		return
	}

	msgID, err := h.max.SendDriverRequestNotification(input.DriverName, message)
	if err != nil {
		log.Printf("miniapp assign: send to DriverRequest failed: %v", err)
	} else if err := h.db.SetRogatkaDriverRequestMessageID(id, msgID); err != nil {
		log.Printf("miniapp assign: save driver message id=%d: %v", id, err)
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

func (h *MiniAppHandler) DeleteRogatkaRequest(w http.ResponseWriter, r *http.Request) {
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

	ok, err := h.db.DeleteRogatkaRequest(id)
	if err != nil {
		http.Error(w, `{"success":false,"error":"Ошибка удаления заявки"}`, http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, `{"success":false,"error":"Заявка не найдена"}`, http.StatusNotFound)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

func (h *MiniAppHandler) CompleteRogatkaRequest(w http.ResponseWriter, r *http.Request) {
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

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, `{"success":false,"error":"Некорректная форма"}`, http.StatusBadRequest)
		return
	}

	driverName := strings.TrimSpace(r.FormValue("driverName"))
	comment := strings.TrimSpace(r.FormValue("comment"))

	if driverName == "" {
		http.Error(w, `{"success":false,"error":"Укажите фамилию водителя"}`, http.StatusBadRequest)
		return
	}
	if len([]rune(comment)) > 2000 {
		http.Error(w, `{"success":false,"error":"Комментарий слишком длинный"}`, http.StatusBadRequest)
		return
	}

	request, err := h.db.GetOpenRogatkaForDriver(id, driverName)
	if err != nil {
		http.Error(w, `{"success":false,"error":"Ошибка загрузки заявки"}`, http.StatusInternalServerError)
		return
	}
	if request == nil {
		http.Error(w, `{"success":false,"error":"Заявка не найдена или уже выполнена"}`, http.StatusNotFound)
		return
	}

	files := r.MultipartForm.File["photos"]
	if len(files) > 5 {
		http.Error(w, `{"success":false,"error":"Можно прикрепить не больше 5 фото"}`, http.StatusBadRequest)
		return
	}

	photos := make([]max2.PhotoUpload, 0, len(files))
	closers := make([]io.Closer, 0, len(files))
	defer func() {
		for _, closer := range closers {
			_ = closer.Close()
		}
	}()

	for _, header := range files {
		ext := strings.ToLower(filepath.Ext(header.Filename))
		switch ext {
		case ".jpg", ".jpeg", ".png", ".webp", ".heic", ".gif":
		default:
			http.Error(w, `{"success":false,"error":"Допустимы только фото"}`, http.StatusBadRequest)
			return
		}

		file, err := header.Open()
		if err != nil {
			http.Error(w, `{"success":false,"error":"Ошибка чтения фото"}`, http.StatusBadRequest)
			return
		}
		closers = append(closers, file)
		photos = append(photos, max2.PhotoUpload{
			Name:   header.Filename,
			Reader: file,
		})
	}

	text := fmt.Sprintf("%s\n\nВодитель: %s", request.Message, driverName)
	if comment != "" {
		text += fmt.Sprintf("\nКомментарий: %s", comment)
	}

	if err := h.max.SendMessageWithPhotos("WorkerDone", text, photos); err != nil {
		log.Printf("miniapp complete: send failed id=%d: %v", id, err)
		http.Error(w, `{"success":false,"error":"Ошибка отправки в группу"}`, http.StatusInternalServerError)
		return
	}

	ok, err := h.db.CompleteRogatkaRequest(id, driverName, comment)
	if err != nil {
		http.Error(w, `{"success":false,"error":"Ошибка сохранения статуса"}`, http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, `{"success":false,"error":"Заявка уже выполнена"}`, http.StatusConflict)
		return
	}

	if request.DriverRequestMessageID != nil && *request.DriverRequestMessageID != "" {
		if err := h.max.MarkDriverRequestCompleted(*request.DriverRequestMessageID, driverName, request.Message); err != nil {
			log.Printf("miniapp complete: edit driver request message id=%d: %v", id, err)
		}
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}
