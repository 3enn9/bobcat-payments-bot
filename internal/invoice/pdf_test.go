package invoice_test

import (
	"bytes"
	"os"
	"testing"
	"time"

	"PaymentsBot/internal/invoice"
)

func TestGeneratePDFSupplierStamps(t *testing.T) {
	for _, tc := range []struct {
		name  string
		inn   string
		stamp bool
	}{
		{"ИП Архипов Николай Николаевич", "645499004617", true},
		{"ИП Архипов Николай Николаевич", "", true},
		{"ИП Архипов Николай Владимирович", "", true},
		{"ИП Архипов Николай Владимирович", "645401925283", true},
		{"ИП Архипов Данила Николаевич", "645407907794", true},
		{"ИП Скрипниченко Алексей Николаевич", "645115810154", true},
		{"ИП Скрипниченко Алексей Николаевич", "", true},
		{"Неизвестный поставщик", "123456789012", false},
		{"Неизвестный поставщик", "", false},
		{"ИП Архипов Данила Николаевич", "", true},
		{`ООО "СарСтройТех"`, "6454116198", true},
		{"ИП Архипов Николай Николаевич", "123456789012", false},
	} {
		t.Run(tc.name+tc.inn, func(t *testing.T) {
			b, err := invoice.GeneratePDF(invoice.PDFData{
				Number: 649, Date: time.Date(2026, 11, 19, 0, 0, 0, 0, time.UTC),
				SupplierName: tc.name, SupplierINN: tc.inn,
				Items: []invoice.PDFItem{{Position: 1, Title: "Услуги", Quantity: 1, Price: 1000, Amount: 1000}},
				Total: 1000,
			})
			if err != nil {
				t.Fatal(err)
			}
			if got := bytes.Contains(b, []byte("/Subtype /Image")); got != tc.stamp {
				t.Fatalf("PDF contains stamp = %v, want %v", got, tc.stamp)
			}
			if tc.inn == "645499004617" {
				if err := os.WriteFile("/tmp/schet_ann_test.pdf", b, 0o644); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

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
