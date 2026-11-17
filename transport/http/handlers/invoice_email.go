package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"PaymentsBot/internal/db"
	"PaymentsBot/internal/invoice"
	mailpkg "PaymentsBot/internal/mail"

	"github.com/gorilla/mux"
)

type invoiceBuyerEmailRequest struct {
	Email string `json:"email"`
}

func (h *MiniAppHandler) ListUnpaidInvoiceEmailFirms(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	firms, err := h.db.ListUnpaidInvoiceEmailFirms()
	if err != nil {
		log.Printf("list unpaid invoice email firms: %v", err)
		http.Error(w, `{"success":false,"error":"Не удалось загрузить фирмы"}`, http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "items": firms})
}

func (h *MiniAppHandler) ListInvoiceBuyerEmails(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	buyerID, ok := invoiceEmailPathID(w, r, "buyerId")
	if !ok {
		return
	}
	emails, err := h.db.ListInvoiceBuyerEmails(buyerID)
	if err != nil {
		log.Printf("list invoice buyer emails buyer=%d: %v", buyerID, err)
		http.Error(w, `{"success":false,"error":"Не удалось загрузить адреса"}`, http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "items": emails})
}

func (h *MiniAppHandler) AddInvoiceBuyerEmail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	buyerID, ok := invoiceEmailPathID(w, r, "buyerId")
	if !ok {
		return
	}
	var input invoiceBuyerEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"success":false,"error":"Некорректный JSON"}`, http.StatusBadRequest)
		return
	}
	input.Email = strings.TrimSpace(input.Email)
	if !mailpkg.IsValidEmail(input.Email) {
		http.Error(w, `{"success":false,"error":"Укажите корректный email"}`, http.StatusBadRequest)
		return
	}

	if err := h.db.AddInvoiceBuyerEmail(buyerID, input.Email); err != nil {
		switch {
		case errors.Is(err, db.ErrInvoiceBuyerNotFound):
			http.Error(w, `{"success":false,"error":"Фирма не найдена среди фирм с неоплаченными счетами"}`, http.StatusNotFound)
		case errors.Is(err, db.ErrInvoiceBuyerEmailExists):
			http.Error(w, `{"success":false,"error":"Этот адрес уже добавлен"}`, http.StatusConflict)
		default:
			log.Printf("add invoice buyer email buyer=%d: %v", buyerID, err)
			http.Error(w, `{"success":false,"error":"Не удалось добавить адрес"}`, http.StatusInternalServerError)
		}
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (h *MiniAppHandler) DeleteInvoiceBuyerEmail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	buyerID, ok := invoiceEmailPathID(w, r, "buyerId")
	if !ok {
		return
	}
	emailID, ok := invoiceEmailPathID(w, r, "emailId")
	if !ok {
		return
	}
	if err := h.db.DeleteInvoiceBuyerEmail(buyerID, emailID); err != nil {
		if errors.Is(err, db.ErrInvoiceBuyerEmailNotFound) {
			http.Error(w, `{"success":false,"error":"Email не найден"}`, http.StatusNotFound)
			return
		}
		log.Printf("delete invoice buyer email buyer=%d email=%d: %v", buyerID, emailID, err)
		http.Error(w, `{"success":false,"error":"Не удалось удалить адрес"}`, http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (h *MiniAppHandler) SendOpenInvoiceEmail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	invoiceID, ok := invoiceEmailPathID(w, r, "id")
	if !ok {
		return
	}
	data, err := h.db.LoadOpenInvoiceEmailData(invoiceID)
	if err != nil {
		switch {
		case errors.Is(err, db.ErrInvoiceEmailNotOpen):
			http.Error(w, `{"success":false,"error":"Счёт уже оплачен, отменён или не найден"}`, http.StatusConflict)
		case errors.Is(err, db.ErrInvoiceNoRecipientEmails):
			http.Error(w, `{"success":false,"error":"Сначала добавьте email фирмы во вкладке «Почта»"}`, http.StatusConflict)
		default:
			log.Printf("load invoice for email invoice=%d: %v", invoiceID, err)
			http.Error(w, `{"success":false,"error":"Не удалось подготовить счёт к отправке"}`, http.StatusInternalServerError)
		}
		return
	}

	pdfItems := make([]invoice.PDFItem, 0, len(data.Items))
	for _, item := range data.Items {
		pdfItems = append(pdfItems, invoice.PDFItem{
			Position: item.Position,
			Title:    item.Title,
			Quantity: item.Quantity,
			Unit:     item.Unit,
			Price:    item.Price,
			Amount:   item.Amount,
		})
	}
	pdfBytes, err := invoice.GeneratePDF(invoice.PDFData{
		Number:          data.Number,
		Date:            data.Date,
		Basis:           data.Basis,
		BankName:        data.BankName,
		BankBIK:         data.BankBIK,
		BankAccount:     data.BankAccount,
		BankCorrAccount: data.BankCorrAccount,
		SupplierName:    data.SupplierName,
		SupplierINN:     data.SupplierINN,
		SupplierKPP:     data.SupplierKPP,
		SupplierAddress: data.SupplierAddress,
		BuyerName:       data.BuyerName,
		BuyerINN:        data.BuyerINN,
		BuyerKPP:        data.BuyerKPP,
		BuyerAddress:    data.BuyerAddress,
		Items:           pdfItems,
		Total:           data.Total,
		VatAmount:       data.VatAmount,
	})
	if err != nil {
		log.Printf("generate invoice email PDF invoice=%d: %v", invoiceID, err)
		http.Error(w, `{"success":false,"error":"Не удалось сформировать PDF счёта"}`, http.StatusInternalServerError)
		return
	}

	fileName := invoice.PDFFileName(data.Number, data.SupplierName, data.SupplierINN, false)
	subject := fmt.Sprintf(
		"Когда ожидать оплату по счёту № %d от %s?",
		data.Number,
		data.Date.Format("02.01.2006"),
	)
	sent := 0
	for _, recipient := range data.Emails {
		if err := h.mail.SendPDF(recipient.Email, subject, fileName, pdfBytes); err != nil {
			log.Printf("send invoice email invoice=%d recipient=%s: %v", invoiceID, recipient.Email, err)
			http.Error(w, `{"success":false,"error":"Не удалось отправить счёт на один из адресов"}`, http.StatusBadGateway)
			return
		}
		sent++
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "sent": sent})
}

func invoiceEmailPathID(w http.ResponseWriter, r *http.Request, key string) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(mux.Vars(r)[key]), 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, `{"success":false,"error":"Некорректный идентификатор"}`, http.StatusBadRequest)
		return 0, false
	}
	return id, true
}
