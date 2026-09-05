package invoice

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestParseItemLine(t *testing.T) {
	line := `1 Услуги экскаватора-погрузчика (ЖК "Лучи") 16 ч 3 000,00 48 000,00`
	item, ok := parseItemLine(line)
	if !ok {
		t.Fatalf("not parsed: %q money=%q", line, reMoneyToken.FindAllString(line, -1))
	}
	if item.Amount != 48000 || item.Price != 3000 || item.Quantity != 16 || item.Unit != "ч" {
		t.Fatalf("%+v", item)
	}
}

func testdata(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no caller")
	}
	p := filepath.Join(filepath.Dir(file), "testdata", name)
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func testdataBytes(t *testing.T, name string) []byte {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no caller")
	}
	b, err := os.ReadFile(filepath.Join(filepath.Dir(file), "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestParseTextSST(t *testing.T) {
	data, err := ParseText(testdata(t, "536_sst.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if data.Number != 536 {
		t.Fatalf("number: %d", data.Number)
	}
	if !data.Date.Equal(time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("date: %s", data.Date)
	}
	if data.SupplierName != `ООО "СарСтройТех"` || data.SupplierINN != "6454116198" || data.SupplierKPP != "645401001" {
		t.Fatalf("supplier: %+v", data)
	}
	if data.BuyerName != `ООО "ППС"ЛЕССТР"` || data.BuyerINN != "6455001697" {
		t.Fatalf("buyer: %s %s", data.BuyerName, data.BuyerINN)
	}
	if data.BankBIK != "044525092" || data.BankAccount != "40702810670010185610" || data.BankCorrAccount != "30101810645250000092" {
		t.Fatalf("bank: %+v", data)
	}
	if data.Total != 69600 || data.VatAmount != 12550.82 {
		t.Fatalf("totals: %v vat=%v", data.Total, data.VatAmount)
	}
	if len(data.Items) != 2 {
		t.Fatalf("items: %+v", data.Items)
	}
	if data.Items[0].Title != `Услуги экскаватора-погрузчика (ЖК "Лучи")` || data.Items[0].Quantity != 16 || data.Items[0].Unit != "ч" || data.Items[0].Amount != 48000 {
		t.Fatalf("item0: %+v", data.Items[0])
	}
	if data.Items[1].Quantity != 9 || data.Items[1].Amount != 21600 {
		t.Fatalf("item1: %+v", data.Items[1])
	}
}

func TestParseTextANN(t *testing.T) {
	data, err := ParseText(testdata(t, "649_ann.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if data.Number != 649 {
		t.Fatalf("number: %d", data.Number)
	}
	if data.SupplierName != "ИП Архипов Николай Николаевич" || data.SupplierINN != "645499004617" || data.SupplierKPP != "" {
		t.Fatalf("supplier: %+v", data)
	}
	if !strings.Contains(data.BuyerName, "Балтийская") || data.BuyerINN != "7801336654" || data.BuyerKPP != "780101001" {
		t.Fatalf("buyer: %s inn=%s kpp=%s", data.BuyerName, data.BuyerINN, data.BuyerKPP)
	}
	if data.BankAccount != "40802810870010171379" {
		t.Fatalf("account: %s", data.BankAccount)
	}
	if !strings.Contains(data.Basis, "Договор на оказание услуг") {
		t.Fatalf("basis: %q", data.Basis)
	}
	if data.Total != 527586 || data.VatAmount != 25123.14 {
		t.Fatalf("totals: %v vat=%v", data.Total, data.VatAmount)
	}
	if len(data.Items) != 2 {
		t.Fatalf("items: %+v", data.Items)
	}
	if data.Items[0].Quantity != 20 || data.Items[0].Unit != "шт" || data.Items[0].Amount != 280000 {
		t.Fatalf("item0: %+v", data.Items[0])
	}
	if data.Items[1].Quantity != 313.4 || data.Items[1].Unit != "т" || data.Items[1].Amount != 247586 {
		t.Fatalf("item1: %+v", data.Items[1])
	}
}

func TestParsePDFSamples(t *testing.T) {
	raw := testdataBytes(t, "536_sst.pdf")
	text, err := ExtractPDFText(raw)
	if err != nil {
		t.Fatal(err)
	}
	sst, err := ParseText(text)
	if err != nil {
		t.Fatalf("sst pdf: %v\n--- extracted ---\n%s", err, text)
	}
	if sst.Number != 536 || sst.Total != 69600 || sst.BuyerINN != "6455001697" {
		t.Fatalf("sst pdf: %+v", sst)
	}

	ann, err := ParsePDF(testdataBytes(t, "649_ann.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	if ann.Number != 649 || ann.Total != 527586 || ann.Items[1].Quantity != 313.4 {
		t.Fatalf("ann pdf: %+v", ann)
	}
}

func TestInvoiceFilename(t *testing.T) {
	if !IsInvoiceFilename(`сч 536 сст.pdf`) || !IsInvoiceFilename(`Сч 649 анн .pdf`) {
		t.Fatal("expected invoice filenames")
	}
	if !IsRevisedFilename(`сч 606 анн изм.pdf`) {
		t.Fatal("expected revised")
	}
	if !IsReconciliationFilename(`Акт сверки ООО ППС.pdf`) {
		t.Fatal("expected act")
	}
	if IsInvoiceFilename(`Акт сверки.pdf`) {
		t.Fatal("act is not invoice")
	}
	if IsInvoiceFilename(`Сч-ф 466 сст .pdf`) || IsInvoiceFilename(`Сч-ф 466 сст изм.pdf`) {
		t.Fatal("factura must not be payment invoice")
	}
}
