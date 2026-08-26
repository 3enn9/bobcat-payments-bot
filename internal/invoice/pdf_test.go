package invoice_test

import (
	"os"
	"testing"
	"time"

	"PaymentsBot/internal/invoice"
)

func TestGeneratePDF(t *testing.T) {
	b, err := invoice.GeneratePDF(invoice.PDFData{
		Number:          536,
		Date:            time.Date(2026, 8, 25, 0, 0, 0, 0, time.Local),
		BankName:        `Московский Филиал АО КБ "Модульбанк" г. Москва`,
		BankBIK:         "044525092",
		BankAccount:     "40702810670010185610",
		BankCorrAccount: "30101810645250000092",
		SupplierName:    `ООО "СарСтройТех"`,
		SupplierINN:     "6454116198",
		SupplierKPP:     "645401001",
		SupplierAddress: "410056, Саратов",
		BuyerName:       `ООО "ППС"ЛЕССТР"`,
		BuyerINN:        "6455001697",
		BuyerKPP:        "645501001",
		BuyerAddress:    "410012, Саратов",
		Items: []invoice.PDFItem{
			{1, "Услуги экскаватора-погрузчика (ЖК \"Лучи\")", 16, "ч", 3000, 48000},
			{2, "Услуги bobcat (вилы, ковш) (ЖК Лучи)", 9, "ч", 2400, 21600},
		},
		Total:     69600,
		VatAmount: 12550.82,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(b) < 1000 {
		t.Fatalf("pdf too small: %d", len(b))
	}
	_ = os.WriteFile("/tmp/schet_test.pdf", b, 0o644)
}
