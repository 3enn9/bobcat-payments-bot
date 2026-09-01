package invoice

import (
	"bytes"
	_ "embed"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
)

//go:embed fonts/Arial.ttf
var fontRegular []byte

//go:embed fonts/Arial-Bold.ttf
var fontBold []byte

type PDFItem struct {
	Position int
	Title    string
	Quantity float64
	Unit     string
	Price    float64
	Amount   float64
}

type PDFData struct {
	Number          int
	Date            time.Time
	Basis           string
	BankName        string
	BankBIK         string
	BankAccount     string
	BankCorrAccount string
	SupplierName    string
	SupplierINN     string
	SupplierKPP     string
	SupplierAddress string
	BuyerName       string
	BuyerINN        string
	BuyerKPP        string
	BuyerAddress    string
	Items           []PDFItem
	Total           float64
	VatAmount       float64
}

func GeneratePDF(data PDFData) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(12, 12, 12)
	pdf.SetAutoPageBreak(true, 14)
	pdf.AddPage()

	pdf.AddUTF8FontFromBytes("arial", "", fontRegular)
	pdf.AddUTF8FontFromBytes("arial", "B", fontBold)

	const pageW = 186.0 // usable width with 12mm margins
	left := 12.0

	// Bank header table
	pdf.SetFont("arial", "", 8)
	rowH := 5.0
	col1 := 95.0
	col2 := 28.0
	col3 := 63.0

	y := 12.0
	pdf.SetFont("arial", "", 8)
	pdf.Rect(left, y, col1, rowH*2, "D")
	pdf.SetXY(left+1, y+1)
	pdf.MultiCell(col1-2, 3.5, data.BankName, "", "L", false)

	pdf.Rect(left+col1, y, col2, rowH, "D")
	pdf.SetXY(left+col1+1, y+0.8)
	pdf.CellFormat(col2-2, 3.5, "БИК", "", 0, "L", false, 0, "")

	pdf.Rect(left+col1+col2, y, col3, rowH, "D")
	pdf.SetXY(left+col1+col2+1, y+0.8)
	pdf.CellFormat(col3-2, 3.5, data.BankBIK, "", 0, "L", false, 0, "")

	pdf.Rect(left+col1, y+rowH, col2, rowH, "D")
	pdf.SetXY(left+col1+1, y+rowH+0.8)
	pdf.CellFormat(col2-2, 3.5, "Сч. №", "", 0, "L", false, 0, "")

	pdf.Rect(left+col1+col2, y+rowH, col3, rowH, "D")
	pdf.SetXY(left+col1+col2+1, y+rowH+0.8)
	pdf.CellFormat(col3-2, 3.5, data.BankCorrAccount, "", 0, "L", false, 0, "")

	y += rowH * 2
	pdf.Rect(left, y, col1, rowH, "D")
	pdf.SetXY(left+1, y+0.8)
	pdf.CellFormat(col1-2, 3.5, "Банк получателя", "", 0, "L", false, 0, "")
	pdf.Rect(left+col1, y, col2+col3, rowH, "D")

	y += rowH
	// Classic 1C layout: INN | KPP on top; name + "Получатель" below; account on the right.
	blockH := rowH * 3
	innW := col1 / 2
	kppW := col1 - innW

	pdf.Rect(left, y, innW, rowH, "D")
	pdf.SetXY(left+1, y+0.8)
	pdf.CellFormat(innW-2, 3.5, fmt.Sprintf("ИНН %s", data.SupplierINN), "", 0, "L", false, 0, "")

	pdf.Rect(left+innW, y, kppW, rowH, "D")
	pdf.SetXY(left+innW+1, y+0.8)
	kppLabel := "КПП"
	if strings.TrimSpace(data.SupplierKPP) != "" {
		kppLabel = fmt.Sprintf("КПП %s", data.SupplierKPP)
	}
	pdf.CellFormat(kppW-2, 3.5, kppLabel, "", 0, "L", false, 0, "")

	pdf.Rect(left, y+rowH, col1, rowH*2, "D")
	pdf.SetXY(left+1, y+rowH+1)
	pdf.SetFont("arial", "", 8)
	pdf.MultiCell(col1-2, 3.5, data.SupplierName, "", "L", false)
	pdf.SetXY(left+1, y+rowH*2+0.8)
	pdf.CellFormat(col1-2, 3.5, "Получатель", "", 0, "L", false, 0, "")

	pdf.Rect(left+col1, y, col2, blockH, "D")
	pdf.SetXY(left+col1+1, y+0.8)
	pdf.CellFormat(col2-2, 3.5, "Сч. №", "", 0, "L", false, 0, "")

	pdf.Rect(left+col1+col2, y, col3, blockH, "D")
	pdf.SetXY(left+col1+col2+1, y+0.8)
	pdf.CellFormat(col3-2, 3.5, data.BankAccount, "", 0, "L", false, 0, "")

	y += blockH + 8
	pdf.SetFont("arial", "B", 13)
	pdf.SetXY(left, y)
	pdf.CellFormat(pageW, 7, fmt.Sprintf("Счет на оплату № %d от %s", data.Number, formatRuDate(data.Date)), "", 1, "L", false, 0, "")
	y = pdf.GetY() + 3

	pdf.SetFont("arial", "", 9)
	writeLabeledBlock := func(label string, value string) {
		pdf.SetXY(left, y)
		pdf.SetFont("arial", "", 8)
		pdf.MultiCell(32, 4, label, "", "L", false)
		labelBottom := pdf.GetY()
		pdf.SetXY(left+34, y)
		pdf.SetFont("arial", "", 9)
		pdf.MultiCell(pageW-34, 4, value, "", "L", false)
		if pdf.GetY() < labelBottom {
			pdf.SetY(labelBottom)
		}
		y = pdf.GetY() + 2
	}

	writeLabeledBlock("Поставщик\n(Исполнитель):", formatPartyLine(
		data.SupplierName, data.SupplierINN, data.SupplierKPP, data.SupplierAddress))
	writeLabeledBlock("Покупатель\n(Заказчик):", formatPartyLine(
		data.BuyerName, data.BuyerINN, data.BuyerKPP, data.BuyerAddress))

	pdf.SetXY(left, y)
	pdf.SetFont("arial", "", 9)
	basis := data.Basis
	if basis == "" {
		basis = "—"
	}
	pdf.CellFormat(pageW, 5, "Основание: "+basis, "", 1, "L", false, 0, "")
	y = pdf.GetY() + 3

	// Items table
	cols := []float64{10, 88, 22, 14, 26, 26}
	headers := []string{"№", "Товары (работы, услуги)", "Кол-во", "Ед.", "Цена", "Сумма"}
	pdf.SetFont("arial", "B", 8)
	x := left
	for i, h := range headers {
		pdf.Rect(x, y, cols[i], 7, "D")
		pdf.SetXY(x, y+1.5)
		pdf.CellFormat(cols[i], 4, h, "", 0, "C", false, 0, "")
		x += cols[i]
	}
	y += 7

	pdf.SetFont("arial", "", 8)
	for _, item := range data.Items {
		qtyText := formatQty(item.Quantity)
		lines := pdf.SplitLines([]byte(item.Title), cols[1]-2)
		rowH := float64(len(lines))*3.6 + 2
		if rowH < 7 {
			rowH = 7
		}
		if y+rowH > 280 {
			pdf.AddPage()
			y = 12
		}
		x = left
		vals := []string{
			fmt.Sprintf("%d", item.Position),
			item.Title,
			qtyText,
			item.Unit,
			formatPDFMoney(item.Price),
			formatPDFMoney(item.Amount),
		}
		aligns := []string{"C", "L", "R", "C", "R", "R"}
		for i, val := range vals {
			pdf.Rect(x, y, cols[i], rowH, "D")
			pdf.SetXY(x+1, y+1.5)
			if i == 1 {
				pdf.MultiCell(cols[i]-2, 3.6, val, "", aligns[i], false)
			} else {
				pdf.CellFormat(cols[i]-2, 3.6, val, "", 0, aligns[i], false, 0, "")
			}
			x += cols[i]
		}
		y += rowH
	}

	y += 2
	totalsX := left + cols[0] + cols[1] + cols[2] + cols[3]
	totalsWLabel := cols[4]
	totalsWValue := cols[5]
	writeTotal := func(label, value string, bold bool) {
		style := ""
		if bold {
			style = "B"
		}
		pdf.SetFont("arial", style, 9)
		pdf.SetXY(totalsX, y)
		pdf.CellFormat(totalsWLabel, 5, label, "", 0, "R", false, 0, "")
		pdf.SetXY(totalsX+totalsWLabel, y)
		pdf.CellFormat(totalsWValue, 5, value, "1", 0, "R", false, 0, "")
		y += 6
	}
	writeTotal("Итого:", formatPDFMoney(data.Total), true)
	if rate := SupplierVATRate(data.SupplierName, data.SupplierINN); rate > 0 {
		writeTotal(fmt.Sprintf("В том числе НДС %d%%:", rate), formatPDFMoney(data.VatAmount), false)
	} else {
		writeTotal("Без НДС:", formatPDFMoney(0), false)
	}
	writeTotal("Всего к оплате:", formatPDFMoney(data.Total), true)

	y += 2
	pdf.SetFont("arial", "", 9)
	pdf.SetXY(left, y)
	pdf.MultiCell(pageW, 4.5, fmt.Sprintf(
		"Всего наименований %d, на сумму %s руб.\n%s",
		len(data.Items),
		formatPDFMoney(data.Total),
		AmountInWords(data.Total),
	), "", "L", false)
	y = pdf.GetY() + 3

	pdf.SetFont("arial", "", 8)
	pdf.SetXY(left, y)
	pdf.MultiCell(pageW, 4, "Оплата данного счета означает согласие с условиями поставки товара.\nУведомление об оплате обязательно, в противном случае не гарантируется наличие товара на складе.\nТовар отпускается по факту прихода денег на р/с Поставщика, самовывозом, при наличии доверенности и паспорта.", "", "L", false)
	y = pdf.GetY() + 10

	pdf.SetFont("arial", "", 10)
	if isEntrepreneurName(data.SupplierName) {
		writeEntrepreneurSignature(pdf, left, y, pageW, data.SupplierName)
	} else {
		pdf.SetXY(left, y)
		pdf.CellFormat(90, 5, "Руководитель ________________  Моисеенко А. Н.", "", 0, "L", false, 0, "")
		pdf.SetXY(left+95, y)
		pdf.CellFormat(90, 5, "Бухгалтер ________________  Моисеенко А. Н.", "", 0, "L", false, 0, "")
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	if pdf.Error() != nil {
		return nil, pdf.Error()
	}
	return buf.Bytes(), nil
}

func writeEntrepreneurSignature(pdf *fpdf.Fpdf, left, y, pageW float64, supplierName string) {
	label := "Предприниматель"
	shortName := entrepreneurShortName(supplierName)

	pdf.SetFont("arial", "", 10)
	labelW := pdf.GetStringWidth(label) + 1
	nameW := pdf.GetStringWidth(shortName) + 1

	lineStart := left + labelW + 3
	lineEnd := left + pageW
	nameX := lineEnd - nameW
	if nameX < lineStart+20 {
		nameX = lineStart + 20
	}

	pdf.SetXY(left, y+2)
	pdf.CellFormat(labelW, 5, label, "", 0, "L", false, 0, "")

	// Фамилия и инициалы справа над линией
	pdf.SetXY(nameX, y)
	pdf.CellFormat(nameW, 4, shortName, "", 0, "L", false, 0, "")

	lineY := y + 5.5
	pdf.SetDrawColor(0, 0, 0)
	pdf.Line(lineStart, lineY, lineEnd, lineY)
}

func isEntrepreneurName(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	n = strings.ReplaceAll(n, "\u00a0", " ")
	return strings.HasPrefix(n, "ип ") || strings.HasPrefix(n, "ип\"") || n == "ип"
}

// SupplierFileCode — короткий код поставщика для имени PDF-файла.
func SupplierFileCode(name, inn string) string {
	n := strings.ToLower(strings.TrimSpace(strings.ReplaceAll(name, "\u00a0", " ")))
	inn = strings.TrimSpace(inn)

	switch {
	case inn == "6454116198" || strings.Contains(n, "сарстройтех"):
		return "сст"
	case strings.Contains(n, "архипов") && (strings.Contains(n, "данила") || strings.Contains(n, "даниил")):
		return "адн"
	case strings.Contains(n, "архипов") && strings.Contains(n, "николай") && strings.Contains(n, "николаевич"):
		return "анн"
	case strings.Contains(n, "архипов") && strings.Contains(n, "николай") && strings.Contains(n, "владимирович"):
		return "анв"
	case strings.Contains(n, "скрипниченко"):
		return "сан"
	default:
		return "проч"
	}
}

// PDFFileName: "сч 536 сст.pdf"
func PDFFileName(number int, supplierName, supplierINN string, revised bool) string {
	code := SupplierFileCode(supplierName, supplierINN)
	if revised {
		return fmt.Sprintf("сч %d %s изм.pdf", number, code)
	}
	return fmt.Sprintf("сч %d %s.pdf", number, code)
}

// entrepreneurShortName: "ИП Архипов Николай Николаевич" → "Архипов Н.Н."
func entrepreneurShortName(full string) string {
	s := strings.TrimSpace(strings.ReplaceAll(full, "\u00a0", " "))
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return s
	}
	if strings.EqualFold(fields[0], "ИП") {
		fields = fields[1:]
	}
	if len(fields) == 0 {
		return s
	}
	surname := fields[0]
	if len(fields) == 1 {
		return surname
	}
	var b strings.Builder
	b.WriteString(surname)
	b.WriteByte(' ')
	for _, part := range fields[1:] {
		runes := []rune(part)
		if len(runes) == 0 {
			continue
		}
		b.WriteRune(runes[0])
		b.WriteByte('.')
	}
	return b.String()
}

// SupplierVATRate — ставка НДС, учтённого в сумме счёта (0 = без НДС).
func SupplierVATRate(name, inn string) int {
	inn = strings.TrimSpace(inn)
	if inn == "6454116198" {
		return 22
	}

	n := strings.ToLower(strings.TrimSpace(strings.ReplaceAll(name, "\u00a0", " ")))
	if strings.Contains(n, "архипов") && (strings.Contains(n, "данила") || strings.Contains(n, "даниил")) {
		return 5
	}

	return 0
}

func CalcVAT(total float64, supplierName, supplierINN string) float64 {
	rate := SupplierVATRate(supplierName, supplierINN)
	if rate == 0 {
		return 0
	}
	return math.Round(total*float64(rate)/float64(100+rate)*100) / 100
}

func formatPartyLine(name, inn, kpp, address string) string {
	parts := []string{name, "ИНН " + inn}
	if strings.TrimSpace(kpp) != "" {
		parts = append(parts, "КПП "+kpp)
	}
	if strings.TrimSpace(address) != "" {
		parts = append(parts, address)
	}
	return strings.Join(parts, ", ")
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
	s := strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.3f", q), "0"), ".")
	return strings.Replace(s, ".", ",", 1)
}

func formatPDFMoney(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	parts := strings.Split(s, ".")
	intPart := parts[0]
	var b strings.Builder
	for i, r := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte(' ')
		}
		b.WriteRune(r)
	}
	return b.String() + "," + parts[1]
}
