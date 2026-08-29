package handlers

import (
	"encoding/json"
	"log"
	"net/http"
)

func (h *MiniAppHandler) ListEquipment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	items, err := h.db.ListEquipment()
	if err != nil {
		log.Printf("list equipment error: %v", err)
		http.Error(w, `{"success":false,"error":"Ошибка загрузки техники"}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"items":   items,
	})
}
