package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"PaymentsBot/internal/db"
)

func (h *MiniAppHandler) ListMatchFirms(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	items, err := h.db.ListMatchFirms()
	if err != nil {
		http.Error(w, `{"success":false,"error":"Ошибка загрузки фирм"}`, http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"items":   items,
	})
}

func (h *MiniAppHandler) ListMatchData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	supplierID, _ := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("supplierId")), 10, 64)
	if supplierID <= 0 {
		http.Error(w, `{"success":false,"error":"Укажите фирму"}`, http.StatusBadRequest)
		return
	}

	payments, err := h.db.ListUnmatchedPaymentsForSupplier(supplierID)
	if err != nil {
		http.Error(w, `{"success":false,"error":"Ошибка загрузки платежей"}`, http.StatusInternalServerError)
		return
	}

	paymentID, _ := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("paymentId")), 10, 64)
	payerINN := ""
	payerName := ""
	if paymentID > 0 {
		found := false
		for _, p := range payments {
			if p.ID == paymentID {
				payerINN = p.PayerINN
				payerName = p.PayerName
				found = true
				break
			}
		}
		if !found {
			http.Error(w, `{"success":false,"error":"Платёж не найден или уже распределён"}`, http.StatusBadRequest)
			return
		}
	}

	var invoices []db.MatchInvoiceItem
	if paymentID > 0 {
		invoices, err = h.db.ListOpenInvoicesForSupplier(supplierID, payerINN, payerName)
		if err != nil {
			http.Error(w, `{"success":false,"error":"Ошибка загрузки счетов"}`, http.StatusInternalServerError)
			return
		}
	} else {
		invoices = []db.MatchInvoiceItem{}
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"payments": payments,
		"invoices": invoices,
	})
}

type matchPaymentRequest struct {
	PaymentID  int64   `json:"paymentId"`
	InvoiceIDs []int64 `json:"invoiceIds"`
}

func (h *MiniAppHandler) MatchPayment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	var input matchPaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"success":false,"error":"Некорректный JSON"}`, http.StatusBadRequest)
		return
	}

	err := h.db.MatchPaymentToInvoices(input.PaymentID, input.InvoiceIDs)
	if err != nil {
		switch {
		case errors.Is(err, db.ErrMatchEmpty):
			http.Error(w, `{"success":false,"error":"Выберите платёж и хотя бы один счёт"}`, http.StatusBadRequest)
		case errors.Is(err, db.ErrPaymentFullyMatched):
			http.Error(w, `{"success":false,"error":"Платёж уже полностью распределён"}`, http.StatusConflict)
		case errors.Is(err, db.ErrInvoiceAlreadyPaid):
			http.Error(w, `{"success":false,"error":"Один из счетов уже полностью оплачен"}`, http.StatusConflict)
		case errors.Is(err, db.ErrMatchForeignInvoice):
			http.Error(w, `{"success":false,"error":"Счёт принадлежит другой фирме"}`, http.StatusBadRequest)
		case errors.Is(err, db.ErrMatchWrongPayer):
			http.Error(w, `{"success":false,"error":"Счёт не относится к плательщику этого платежа"}`, http.StatusBadRequest)
		default:
			http.Error(w, `{"success":false,"error":"Не удалось сопоставить"}`, http.StatusInternalServerError)
		}
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}
