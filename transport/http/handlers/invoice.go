package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"PaymentsBot/internal/db"
	"PaymentsBot/internal/invoice"
	mailpkg "PaymentsBot/internal/mail"
	max2 "PaymentsBot/internal/max"
)

type partyPayload struct {
	ID          *int64 `json:"id"`
	Name        string `json:"name"`
	INN         string `json:"inn"`
	KPP         string `json:"kpp"`
	AddressText string `json:"addressText"`
	Email       string `json:"email"`
}

type bankPayload struct {
	ID          *int64 `json:"id"`
	Name        string `json:"name"`
	BIK         string `json:"bik"`
	Account     string `json:"account"`
	CorrAccount string `json:"corrAccount"`
}

type itemPayload struct {
	Position int     `json:"position"`
	Title    string  `json:"title"`
	Quantity float64 `json:"quantity"`
	Unit     string  `json:"unit"`
	Price    float64 `json:"price"`
	Amount   float64 `json:"amount"`
}

type createInvoiceRequest struct {
	Number      int           `json:"number"`
	InvoiceDate string        `json:"invoiceDate"`
	Basis       string        `json:"basis"`
	Supplier    partyPayload  `json:"supplier"`
	Buyer       partyPayload  `json:"buyer"`
	Bank        bankPayload   `json:"bank"`
	Items       []itemPayload `json:"items"`
	SendToEmail      bool          `json:"sendToEmail"`
	ReplaceExisting  bool          `json:"replaceExisting"`
}

func (h *MiniAppHandler) SearchInvoiceSuppliers(w http.ResponseWriter, r *http.Request) {
	h.searchJSON(w, r, func(q string) (any, error) {
		return h.db.SearchInvoiceSuppliers(q, 20)
	})
}

func (h *MiniAppHandler) SearchInvoiceBuyers(w http.ResponseWriter, r *http.Request) {
	h.searchJSON(w, r, func(q string) (any, error) {
		return h.db.SearchInvoiceBuyers(q, 20)
	})
}

func (h *MiniAppHandler) SearchInvoiceBanks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	supplierID, _ := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("supplierId")), 10, 64)
	items, err := h.db.SearchInvoiceBanks(q, supplierID, 20)
	if err != nil {
		http.Error(w, `{"success":false,"error":"Ошибка поиска"}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"items":   items,
	})
}

func (h *MiniAppHandler) searchJSON(w http.ResponseWriter, r *http.Request, fn func(q string) (any, error)) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	items, err := fn(strings.TrimSpace(r.URL.Query().Get("q")))
	if err != nil {
		http.Error(w, `{"success":false,"error":"Ошибка поиска"}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"items":   items,
	})
}

func (h *MiniAppHandler) CreateInvoice(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	var input createInvoiceRequest
	photos := make([]max2.PhotoUpload, 0)
	closers := make([]io.Closer, 0)
	defer func() {
		for _, closer := range closers {
			_ = closer.Close()
		}
	}()

	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			http.Error(w, `{"success":false,"error":"Некорректная форма"}`, http.StatusBadRequest)
			return
		}
		raw := strings.TrimSpace(r.FormValue("payload"))
		if raw == "" {
			http.Error(w, `{"success":false,"error":"Некорректный JSON"}`, http.StatusBadRequest)
			return
		}
		if err := json.Unmarshal([]byte(raw), &input); err != nil {
			http.Error(w, `{"success":false,"error":"Некорректный JSON"}`, http.StatusBadRequest)
			return
		}

		files := r.MultipartForm.File["photos"]
		if len(files) > 10 {
			http.Error(w, `{"success":false,"error":"Можно прикрепить не больше 10 фото"}`, http.StatusBadRequest)
			return
		}
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
	} else if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"success":false,"error":"Некорректный JSON"}`, http.StatusBadRequest)
		return
	}

	input.Basis = strings.TrimSpace(input.Basis)
	input.Supplier.Name = strings.TrimSpace(input.Supplier.Name)
	input.Supplier.INN = strings.TrimSpace(input.Supplier.INN)
	input.Supplier.KPP = strings.TrimSpace(input.Supplier.KPP)
	input.Supplier.AddressText = strings.TrimSpace(input.Supplier.AddressText)
	input.Buyer.Name = strings.TrimSpace(input.Buyer.Name)
	input.Buyer.INN = strings.TrimSpace(input.Buyer.INN)
	input.Buyer.KPP = strings.TrimSpace(input.Buyer.KPP)
	input.Buyer.AddressText = strings.TrimSpace(input.Buyer.AddressText)
	input.Buyer.Email = strings.TrimSpace(input.Buyer.Email)
	input.Bank.Name = strings.TrimSpace(input.Bank.Name)
	input.Bank.BIK = strings.TrimSpace(input.Bank.BIK)
	input.Bank.Account = strings.TrimSpace(input.Bank.Account)
	input.Bank.CorrAccount = strings.TrimSpace(input.Bank.CorrAccount)

	if input.Supplier.Name == "" || input.Supplier.INN == "" || input.Supplier.AddressText == "" {
		http.Error(w, `{"success":false,"error":"Заполните данные поставщика"}`, http.StatusBadRequest)
		return
	}
	if input.Buyer.Name == "" || input.Buyer.INN == "" || input.Buyer.AddressText == "" {
		http.Error(w, `{"success":false,"error":"Заполните данные покупателя"}`, http.StatusBadRequest)
		return
	}
	if input.Bank.Name == "" || input.Bank.BIK == "" || input.Bank.Account == "" || input.Bank.CorrAccount == "" {
		http.Error(w, `{"success":false,"error":"Заполните данные банка"}`, http.StatusBadRequest)
		return
	}
	if len(input.Items) == 0 {
		http.Error(w, `{"success":false,"error":"Добавьте хотя бы одну позицию"}`, http.StatusBadRequest)
		return
	}
	if input.Number <= 0 {
		http.Error(w, `{"success":false,"error":"Укажите номер счёта"}`, http.StatusBadRequest)
		return
	}
	if input.SendToEmail {
		if input.Buyer.Email == "" {
			http.Error(w, `{"success":false,"error":"Укажите email для отправки"}`, http.StatusBadRequest)
			return
		}
		if !mailpkg.IsValidEmail(input.Buyer.Email) {
			http.Error(w, `{"success":false,"error":"Некорректный email"}`, http.StatusBadRequest)
			return
		}
	}

	invoiceDate, err := time.Parse("2006-01-02", strings.TrimSpace(input.InvoiceDate))
	if err != nil {
		http.Error(w, `{"success":false,"error":"Некорректная дата счёта"}`, http.StatusBadRequest)
		return
	}

	items := make([]db.InvoiceItemInput, 0, len(input.Items))
	var total float64
	for i, item := range input.Items {
		title := strings.TrimSpace(item.Title)
		unit := strings.TrimSpace(item.Unit)
		if title == "" || unit == "" || item.Quantity <= 0 || item.Price < 0 {
			http.Error(w, `{"success":false,"error":"Проверьте позиции счёта"}`, http.StatusBadRequest)
			return
		}
		amount := roundMoney(item.Quantity * item.Price)
		if item.Amount > 0 {
			amount = roundMoney(item.Amount)
		}
		position := item.Position
		if position <= 0 {
			position = i + 1
		}
		items = append(items, db.InvoiceItemInput{
			Position: position,
			Title:    title,
			Quantity: item.Quantity,
			Unit:     unit,
			Price:    roundMoney(item.Price),
			Amount:   amount,
		})
		total += amount
	}
	total = roundMoney(total)
	vat := invoice.CalcVAT(total, input.Supplier.INN)

	created, err := h.db.CreateInvoice(db.CreateInvoiceInput{
		SupplierID:      input.Supplier.ID,
		BuyerID:         input.Buyer.ID,
		BankID:          input.Bank.ID,
		Number:          input.Number,
		ReplaceExisting: input.ReplaceExisting,
		InvoiceDate:     invoiceDate,
		Basis:           input.Basis,
		SupplierName:    input.Supplier.Name,
		SupplierINN:     input.Supplier.INN,
		SupplierKPP:     input.Supplier.KPP,
		SupplierAddress: input.Supplier.AddressText,
		BankName:        input.Bank.Name,
		BankBIK:         input.Bank.BIK,
		BankAccount:     input.Bank.Account,
		BankCorrAccount: input.Bank.CorrAccount,
		BuyerName:       input.Buyer.Name,
		BuyerINN:        input.Buyer.INN,
		BuyerKPP:        input.Buyer.KPP,
		BuyerAddress:    input.Buyer.AddressText,
		BuyerEmail:      input.Buyer.Email,
		Total:           total,
		VatAmount:       vat,
		Items:           items,
	})
	if err != nil {
		log.Printf("create invoice error: %v", err)
		if errors.Is(err, db.ErrInvoiceExists) {
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"code":    "invoice_exists",
				"error":   fmt.Sprintf("Счёт № %d уже существует. Заменить его новым?", input.Number),
			})
			return
		}
		http.Error(w, `{"success":false,"error":"Ошибка сохранения счёта"}`, http.StatusInternalServerError)
		return
	}

	pdfItems := make([]invoice.PDFItem, 0, len(items))
	for _, item := range items {
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
		Number:          created.Number,
		Date:            invoiceDate,
		Basis:           input.Basis,
		BankName:        input.Bank.Name,
		BankBIK:         input.Bank.BIK,
		BankAccount:     input.Bank.Account,
		BankCorrAccount: input.Bank.CorrAccount,
		SupplierName:    input.Supplier.Name,
		SupplierINN:     input.Supplier.INN,
		SupplierKPP:     input.Supplier.KPP,
		SupplierAddress: input.Supplier.AddressText,
		BuyerName:       input.Buyer.Name,
		BuyerINN:        input.Buyer.INN,
		BuyerKPP:        input.Buyer.KPP,
		BuyerAddress:    input.Buyer.AddressText,
		Items:           pdfItems,
		Total:           total,
		VatAmount:       vat,
	})
	if err != nil {
		log.Printf("invoice pdf error: %v", err)
		http.Error(w, `{"success":false,"error":"Ошибка формирования PDF"}`, http.StatusInternalServerError)
		return
	}

	fileName := invoice.PDFFileName(created.Number, input.Supplier.Name, input.Supplier.INN, created.Replaced)
	if err := h.max.SendFileAndPhotosToGroup("Invoices", fileName, bytes.NewReader(pdfBytes), photos); err != nil {
		log.Printf("invoice send to MAX failed: %v", err)
		http.Error(w, `{"success":false,"error":"Ошибка отправки в группу"}`, http.StatusInternalServerError)
		return
	}

	if input.SendToEmail {
		log.Printf("invoice email stub: счёт № %d -> %s (отправка на почту пока отключена)", created.Number, input.Buyer.Email)
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"id":       created.ID,
		"number":   created.Number,
		"replaced": created.Replaced,
	})
}

func roundMoney(v float64) float64 {
	return math.Round(v*100) / 100
}
