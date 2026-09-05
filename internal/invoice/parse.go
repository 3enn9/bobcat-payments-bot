package invoice

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

var (
	reInvoiceHeader = regexp.MustCompile(`(?i)Счет на оплату\s*№\s*(\d+)\s+от\s+(\d{1,2})\s+(\p{L}+)\s+(\d{4})`)
	reBIK           = regexp.MustCompile(`БИК\s+(\d{9})`)
	reAccount       = regexp.MustCompile(`Сч\.\s*№\s*(\d{20})`)
	reINN           = regexp.MustCompile(`ИНН\s+(\d{10,12})`)
	reKPP           = regexp.MustCompile(`КПП\s+(\d{9})`)
	reTotal         = regexp.MustCompile(`(?i)Итого:\s*([\d\s\x{00a0}]+,\d{2})`)
	rePayTotal      = regexp.MustCompile(`(?i)Всего к оплате:\s*([\d\s\x{00a0}]+,\d{2})`)
	reVAT           = regexp.MustCompile(`(?i)(?:В том числе\s+)?НДС[^:]*:\s*([\d\s\x{00a0}]+,\d{2})`)
	reSupplierBlock = regexp.MustCompile(`(?is)Поставщик\s+(.*?)\s+Покупатель`)
	reBuyerBlock    = regexp.MustCompile(`(?is)Покупатель\s+(.*?)\s+Основание:`)
	reBasis         = regexp.MustCompile(`(?is)Основание:\s*(.*?)\s*(?:№\s*Товары|Товары \(работы)`)
	reItemLine      = regexp.MustCompile(`(?m)^\s*(\d{1,3})\s+(\S.*)$`)
	reMoneyToken    = regexp.MustCompile(`\d{1,3}(?:[\s\x{00a0}]\d{3})+,\d{2}|\d{1,3},\d{2}`)
	rePosTitle      = regexp.MustCompile(`^(\d{1,3})\s+(.*)$`)
)

var ruMonths = map[string]time.Month{
	"января": 1, "февраля": 2, "марта": 3, "апреля": 4,
	"мая": 5, "июня": 6, "июля": 7, "августа": 8,
	"сентября": 9, "октября": 10, "ноября": 11, "декабря": 12,
}

// IsInvoiceFilename — вложение вида «сч 536 сст.pdf» / «Сч 649 анн изм.pdf».
// Счета-фактуры (Сч-ф …) не считаем счетами на оплату.
func IsInvoiceFilename(name string) bool {
	n := strings.ToLower(strings.TrimSpace(strings.ReplaceAll(name, "\u00a0", " ")))
	n = strings.Trim(n, `"'`)
	if n == "" || !strings.HasSuffix(n, ".pdf") {
		return false
	}
	if isFacturaFilename(n) {
		return false
	}
	return strings.Contains(n, "сч ") || strings.HasPrefix(n, "сч ")
}

func isFacturaFilename(n string) bool {
	n = strings.ToLower(n)
	return strings.Contains(n, "сч-ф") ||
		strings.Contains(n, "счф") ||
		strings.Contains(n, "счет-фактур") ||
		strings.Contains(n, "счёт-фактур") ||
		strings.Contains(n, "фактур")
}

func IsReconciliationFilename(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	return strings.Contains(n, "акт сверки")
}

func IsRevisedFilename(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	return strings.Contains(n, "изм")
}

func ParsePDF(raw []byte) (*PDFData, error) {
	text, err := ExtractPDFText(raw)
	if err != nil {
		return nil, err
	}
	return ParseText(text)
}

func ParseText(text string) (*PDFData, error) {
	text = normalizeInvoiceText(text)
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("пустой текст счёта")
	}

	header := reInvoiceHeader.FindStringSubmatch(text)
	if header == nil {
		return nil, fmt.Errorf("не найден заголовок «Счет на оплату»")
	}
	number, err := strconv.Atoi(header[1])
	if err != nil || number <= 0 {
		return nil, fmt.Errorf("некорректный номер счёта")
	}
	day, _ := strconv.Atoi(header[2])
	month, ok := ruMonths[strings.ToLower(header[3])]
	if !ok {
		return nil, fmt.Errorf("неизвестный месяц: %s", header[3])
	}
	year, _ := strconv.Atoi(header[4])
	date := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)

	bikMatch := reBIK.FindStringSubmatch(text)
	if bikMatch == nil {
		return nil, fmt.Errorf("не найден БИК")
	}
	accounts := reAccount.FindAllStringSubmatch(text, 2)
	if len(accounts) < 2 {
		return nil, fmt.Errorf("не найдены расчётный и корр. счета")
	}

	bankName := extractBankName(text)
	supplier, err := parsePartyBlock(reSupplierBlock, text, "(Исполнитель):")
	if err != nil {
		return nil, fmt.Errorf("поставщик: %w", err)
	}
	buyer, err := parsePartyBlock(reBuyerBlock, text, "(Заказчик):")
	if err != nil {
		return nil, fmt.Errorf("покупатель: %w", err)
	}

	basis := ""
	if m := reBasis.FindStringSubmatch(text); m != nil {
		basis = collapseSpace(strings.TrimSpace(m[1]))
		if basis == "—" || basis == "-" {
			basis = ""
		}
	}

	items, err := parseItems(text)
	if err != nil {
		return nil, err
	}

	total := 0.0
	if m := rePayTotal.FindStringSubmatch(text); m != nil {
		total = mustMoney(m[1])
	} else if m := reTotal.FindStringSubmatch(text); m != nil {
		total = mustMoney(m[1])
	}
	if total == 0 {
		for _, item := range items {
			total += item.Amount
		}
		total = roundMoney(total)
	}

	vat := 0.0
	if m := reVAT.FindStringSubmatch(text); m != nil {
		vat = mustMoney(m[1])
	}

	return &PDFData{
		Number:          number,
		Date:            date,
		Basis:           basis,
		BankName:        bankName,
		BankBIK:         bikMatch[1],
		BankAccount:     accounts[1][1],
		BankCorrAccount: accounts[0][1],
		SupplierName:    supplier.name,
		SupplierINN:     supplier.inn,
		SupplierKPP:     supplier.kpp,
		SupplierAddress: supplier.address,
		BuyerName:       buyer.name,
		BuyerINN:        buyer.inn,
		BuyerKPP:        buyer.kpp,
		BuyerAddress:    buyer.address,
		Items:           items,
		Total:           total,
		VatAmount:       vat,
	}, nil
}

type parsedParty struct {
	name    string
	inn     string
	kpp     string
	address string
}

func parsePartyBlock(re *regexp.Regexp, text, skipLabel string) (parsedParty, error) {
	m := re.FindStringSubmatch(text)
	if m == nil {
		return parsedParty{}, fmt.Errorf("блок не найден")
	}
	raw := collapseSpace(m[1])
	raw = strings.ReplaceAll(raw, skipLabel, " ")
	raw = collapseSpace(raw)

	innMatch := reINN.FindStringSubmatch(raw)
	if innMatch == nil {
		return parsedParty{}, fmt.Errorf("нет ИНН")
	}
	kpp := ""
	if km := reKPP.FindStringSubmatch(raw); km != nil {
		kpp = km[1]
	}

	namePart, rest, ok := strings.Cut(raw, ", ИНН")
	if !ok {
		return parsedParty{}, fmt.Errorf("нет имени до ИНН")
	}
	name := strings.TrimSpace(namePart)
	rest = collapseSpace("ИНН" + rest)

	addr := rest
	if idx := strings.Index(addr, innMatch[1]); idx >= 0 {
		addr = strings.TrimSpace(addr[idx+len(innMatch[1]):])
	}
	addr = strings.TrimSpace(strings.TrimPrefix(addr, ","))
	if kpp != "" {
		addr = strings.TrimSpace(strings.TrimPrefix(addr, "КПП "+kpp))
		addr = strings.TrimSpace(strings.TrimPrefix(addr, ","))
	}

	if name == "" || innMatch[1] == "" {
		return parsedParty{}, fmt.Errorf("пустые реквизиты")
	}
	return parsedParty{name: name, inn: innMatch[1], kpp: kpp, address: addr}, nil
}

func parseItems(text string) ([]PDFItem, error) {
	start := strings.Index(text, "Товары (работы, услуги)")
	if start < 0 {
		return nil, fmt.Errorf("не найдена таблица позиций")
	}
	body := text[start:]
	endMarkers := []string{"\nИтого:", "\nИтого ", "Итого:"}
	end := len(body)
	for _, m := range endMarkers {
		if i := strings.Index(body, m); i > 0 && i < end {
			end = i
		}
	}
	body = body[:end]

	var items []PDFItem
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "№") || strings.Contains(line, "Товары (работы") {
			continue
		}
		item, ok := parseItemLine(line)
		if !ok {
			continue
		}
		items = append(items, item)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("позиции не распознаны")
	}
	return items, nil
}

func parseItemLine(line string) (PDFItem, bool) {
	line = strings.TrimSpace(line)
	if !reItemLine.MatchString(line) {
		return PDFItem{}, false
	}

	money := reMoneyToken.FindAllStringIndex(line, -1)
	if len(money) < 2 {
		return PDFItem{}, false
	}
	amountSpan := money[len(money)-1]
	priceSpan := money[len(money)-2]
	if strings.TrimSpace(line[amountSpan[1]:]) != "" {
		return PDFItem{}, false
	}

	amount := mustMoney(line[amountSpan[0]:amountSpan[1]])
	price := mustMoney(line[priceSpan[0]:priceSpan[1]])
	rest := strings.TrimSpace(line[:priceSpan[0]])

	fields := strings.Fields(rest)
	if len(fields) < 3 {
		return PDFItem{}, false
	}
	unit := fields[len(fields)-1]
	qty := parseQty(fields[len(fields)-2])
	if qty <= 0 {
		return PDFItem{}, false
	}
	rest = strings.Join(fields[:len(fields)-2], " ")

	m := rePosTitle.FindStringSubmatch(rest)
	if m == nil {
		return PDFItem{}, false
	}
	pos, err := strconv.Atoi(m[1])
	if err != nil || pos <= 0 {
		return PDFItem{}, false
	}
	title := strings.TrimSpace(m[2])
	if title == "" {
		return PDFItem{}, false
	}
	return PDFItem{
		Position: pos,
		Title:    title,
		Quantity: qty,
		Unit:     unit,
		Price:    price,
		Amount:   amount,
	}, true
}

func extractBankName(text string) string {
	head := text
	if i := strings.Index(text, "Банк получателя"); i > 0 {
		head = text[:i]
	}
	if i := strings.Index(head, "БИК"); i > 0 {
		head = head[:i]
	}
	return strings.TrimSpace(collapseSpace(head))
}

func normalizeInvoiceText(text string) string {
	text = strings.ReplaceAll(text, "\u00a0", " ")
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRightFunc(collapseSpace(line), unicode.IsSpace)
	}
	return strings.Join(lines, "\n")
}

func collapseSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func parseQty(s string) float64 {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", "."))
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func mustMoney(s string) float64 {
	s = strings.ReplaceAll(s, "\u00a0", "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, ",", ".")
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func roundMoney(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}
