package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"PaymentsBot/internal/db"
	"PaymentsBot/internal/invoice"
)

type partyPayload struct {
	ID          *int64 `json:"id"`
	Name        string `json:"name"`
	INN         string `json:"inn"`
	KPP         string `json:"kpp"`
	AddressText string `json:"addressText"`
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
	InvoiceDate string        `json:"invoiceDate"`
	Basis       string        `json:"basis"`
	Supplier    partyPayload  `json:"supplier"`
	Buyer       partyPayload  `json:"buyer"`
	Bank        bankPayload   `json:"bank"`
	Items       []itemPayload `json:"items"`
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
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
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
	vat := roundMoney(total * 22 / 122)

	created, err := h.db.CreateInvoice(db.CreateInvoiceInput{
		SupplierID:      input.Supplier.ID,
		BuyerID:         input.Buyer.ID,
		BankID:          input.Bank.ID,
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
		Total:           total,
		VatAmount:       vat,
		Items:           items,
	})
	if err != nil {
		log.Printf("create invoice error: %v", err)
		http.Error(w, `{"success":false,"error":"Ошибка сохранения счёта"}`, http.StatusInternalServerError)
		return
	}

	text := formatInvoiceMessage(created.Number, invoiceDate, input.Basis, input.Supplier, input.Buyer, input.Bank, items, total, vat)
	if err := h.max.SendMessageInGroupName("Invoices", text); err != nil {
		log.Printf("invoice send to MAX failed: %v", err)
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"id":      created.ID,
		"number":  created.Number,
	})
}

func roundMoney(v float64) float64 {
	return math.Round(v*100) / 100
}

func formatInvoiceMessage(
	number int,
	date time.Time,
	basis string,
	supplier partyPayload,
	buyer partyPayload,
	bank bankPayload,
	items []db.InvoiceItemInput,
	total, vat float64,
) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s\nБанк получателя\n", bank.Name))
	b.WriteString(fmt.Sprintf("БИК %s\n", bank.BIK))
	b.WriteString(fmt.Sprintf("Корр. счёт %s\n", bank.CorrAccount))
	b.WriteString(fmt.Sprintf("Р/с %s\n\n", bank.Account))
	b.WriteString(fmt.Sprintf("Счет на оплату № %d от %s\n\n", number, formatRuDate(date)))
	b.WriteString(fmt.Sprintf(
		"Поставщик (Исполнитель):\n%s, ИНН %s, КПП %s, %s\n\n",
		supplier.Name, supplier.INN, supplier.KPP, supplier.AddressText,
	))
	b.WriteString(fmt.Sprintf(
		"Покупатель (Заказчик):\n%s, ИНН %s, КПП %s, %s\n\n",
		buyer.Name, buyer.INN, buyer.KPP, buyer.AddressText,
	))
	if basis != "" {
		b.WriteString(fmt.Sprintf("Основание: %s\n\n", basis))
	}
	b.WriteString("№ | Товары (работы, услуги) | Кол-во | Ед. | Цена | Сумма\n")
	for _, item := range items {
		b.WriteString(fmt.Sprintf(
			"%d | %s | %s %s | %s | %s\n",
			item.Position,
			item.Title,
			formatQty(item.Quantity),
			item.Unit,
			db.FormatMoney(item.Price),
			db.FormatMoney(item.Amount),
		))
	}
	b.WriteString(fmt.Sprintf("\nИтого: %s\n", db.FormatMoney(total)))
	b.WriteString(fmt.Sprintf("В том числе НДС 22%%: %s\n", db.FormatMoney(vat)))
	b.WriteString(fmt.Sprintf("Всего к оплате: %s\n\n", db.FormatMoney(total)))
	b.WriteString(fmt.Sprintf(
		"Всего наименований %d, на сумму %s руб.\n%s",
		len(items),
		db.FormatMoney(total),
		invoice.AmountInWords(total),
	))
	return b.String()
}

func formatRuDate(t time.Time) string {
	months := []string{
		"", "января", "февраля", "марта", "апреля", "мая", "июня",
		"июля", "августа", "сентября", "октября", "ноября", "декабря",
	}
	return fmt.Sprintf("%d %s %d г.", t.Day(), months[int(t.Month())], t.Year())
}

func formatQty(q float64) string {
	if q == float64(int64(q)) {
		return fmt.Sprintf("%d", int64(q))
	}
	return strings.Replace(fmt.Sprintf("%.3f", q), ".", ",", 1)
}
