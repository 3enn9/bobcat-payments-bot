package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"PaymentsBot/internal/db"

	"github.com/gorilla/mux"
)

type MiniAppHandler struct {
	db *db.Database
}

func NewMiniAppHandler(database *db.Database) *MiniAppHandler {
	return &MiniAppHandler{
		db: database,
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

	requests, err := h.db.ListRogatkaRequests()
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

	ok, err := h.db.AssignRogatkaDriver(id, input.DriverName)
	if err != nil {
		http.Error(w, `{"success":false,"error":"Ошибка назначения водителя"}`, http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, `{"success":false,"error":"Заявка не найдена или уже назначена"}`, http.StatusNotFound)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}
