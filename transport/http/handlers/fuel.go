package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"PaymentsBot/internal/clock"
	"PaymentsBot/internal/db"

	"github.com/gorilla/mux"
)

func (h *MiniAppHandler) ListFuelEntries(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	holder := strings.TrimSpace(r.URL.Query().Get("holder"))
	if holder == "" {
		http.Error(w, `{"success":false,"error":"Укажите носителя карты"}`, http.StatusBadRequest)
		return
	}

	since := clock.Now().AddDate(0, 0, -30)
	items, err := h.db.ListFuelEntriesByHolder(holder, since)
	if err != nil {
		log.Printf("list fuels error: %v", err)
		http.Error(w, `{"success":false,"error":"Ошибка загрузки заправок"}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"items":   items,
	})
}

type updateFuelRequest struct {
	EquipmentNumber string `json:"equipmentNumber"`
	FuelKind        string `json:"fuelKind"`
}

func (h *MiniAppHandler) UpdateFuelEquipment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	id, _ := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if id <= 0 {
		http.Error(w, `{"success":false,"error":"Заправка не найдена"}`, http.StatusBadRequest)
		return
	}

	var input updateFuelRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"success":false,"error":"Некорректный JSON"}`, http.StatusBadRequest)
		return
	}

	number := strings.TrimSpace(input.EquipmentNumber)
	if len([]rune(number)) > 32 {
		http.Error(w, `{"success":false,"error":"Номер техники слишком длинный"}`, http.StatusBadRequest)
		return
	}

	kind, ok := db.ParseFuelKind(input.FuelKind)
	if !ok {
		http.Error(w, `{"success":false,"error":"Укажите бензин или ДТ"}`, http.StatusBadRequest)
		return
	}

	if err := h.db.UpdateFuelEquipment(id, number, kind); err != nil {
		log.Printf("update fuel equipment error: %v", err)
		http.Error(w, `{"success":false,"error":"Не удалось сохранить номер"}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

type splitFuelRequest struct {
	Parts []db.FuelSplitPart `json:"parts"`
}

func (h *MiniAppHandler) SplitFuelEntry(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	id, _ := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if id <= 0 {
		http.Error(w, `{"success":false,"error":"Заправка не найдена"}`, http.StatusBadRequest)
		return
	}

	var input splitFuelRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"success":false,"error":"Некорректный JSON"}`, http.StatusBadRequest)
		return
	}

	if err := h.db.SplitFuelEntry(id, input.Parts); err != nil {
		log.Printf("split fuel error: %v", err)
		http.Error(w, fmt.Sprintf(`{"success":false,"error":%q}`, err.Error()), http.StatusBadRequest)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}
