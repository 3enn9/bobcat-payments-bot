package invoice

import (
	"bytes"
	"fmt"
	"time"

	"github.com/go-pdf/fpdf"
)

type UnpaidFirmTable struct {
	FirmName string
	FirmINN  string
	Rows     []UnpaidTableRow
}

type UnpaidTableRow struct {
	Number    int
	Date      time.Time
	BuyerName string
	Total     float64
	Paid      float64
	Remaining float64
}

// GenerateUnpaidTablesPDF — landscape A4 tables of unpaid invoices, one section per firm.
func GenerateUnpaidTablesPDF(firms []UnpaidFirmTable, generatedAt time.Time) ([]byte, error) {
	pdf := fpdf.New("L", "mm", "A4", "")
	pdf.SetMargins(10, 10, 10)
	pdf.SetAutoPageBreak(true, 12)
	pdf.AddUTF8FontFromBytes("arial", "", fontRegular)
	pdf.AddUTF8FontFromBytes("arial", "B", fontBold)

	const pageW = 277.0 // A4 landscape usable width with 10mm margins
	left := 10.0

	// columns: № | дата | покупатель | сумма | оплачено | остаток
	colN := 12.0
	colDate := 20.0
	colBuyer := 72.0
	colTotal := 30.0
	colPaid := 30.0
	colRem := 30.0
	rowH := 7.0

	writeHeader := func(firm UnpaidFirmTable) {
		pdf.SetFont("arial", "B", 12)
		title := firm.FirmName
		if firm.FirmINN != "" {
			title += "  ·  ИНН " + firm.FirmINN
		}
		pdf.SetXY(left, pdf.GetY())
		pdf.MultiCell(pageW, 6, title, "", "L", false)
		pdf.Ln(1)

		pdf.SetFont("arial", "", 9)
		pdf.SetXY(left, pdf.GetY())
		pdf.CellFormat(pageW, 5, fmt.Sprintf("Неоплаченные счета · %s · %d шт.", generatedAt.Format("02.01.2006 15:04"), len(firm.Rows)), "", 1, "L", false, 0, "")
		pdf.Ln(2)

		pdf.SetFont("arial", "B", 8)
		pdf.SetFillColor(235, 235, 235)
		y := pdf.GetY()
		writeCell := func(x, w float64, text, align string) {
			pdf.Rect(x, y, w, rowH, "FD")
			pdf.SetXY(x+1, y+1.5)
			pdf.CellFormat(w-2, 4, text, "", 0, align, false, 0, "")
		}
		x := left
		writeCell(x, colN, "№", "C")
		x += colN
		writeCell(x, colDate, "Дата", "C")
		x += colDate
		writeCell(x, colBuyer, "Покупатель", "L")
		x += colBuyer
		writeCell(x, colTotal, "Сумма", "R")
		x += colTotal
		writeCell(x, colPaid, "Оплачено", "R")
		x += colPaid
		writeCell(x, colRem, "Остаток", "R")
		pdf.SetY(y + rowH)
		pdf.SetFillColor(255, 255, 255)
	}

	if len(firms) == 0 {
		pdf.AddPage()
		pdf.SetFont("arial", "B", 14)
		pdf.SetXY(left, 40)
		pdf.CellFormat(pageW, 8, "Неоплаченных счетов нет", "", 1, "C", false, 0, "")
		pdf.SetFont("arial", "", 10)
		pdf.CellFormat(pageW, 6, generatedAt.Format("02.01.2006 15:04"), "", 1, "C", false, 0, "")
	}

	for _, firm := range firms {
		pdf.AddPage()
		writeHeader(firm)

		pdf.SetFont("arial", "", 8)
		var sumTotal, sumPaid, sumRem float64
		for _, row := range firm.Rows {
			if pdf.GetY()+rowH > 200 {
				pdf.AddPage()
				writeHeader(firm)
				pdf.SetFont("arial", "", 8)
			}
			y := pdf.GetY()
			buyer := truncateRunes(ShortenBuyerName(row.BuyerName), 42)
			date := "—"
			if !row.Date.IsZero() {
				date = row.Date.Format("02.01.06")
			}
			x := left
			draw := func(w float64, text, align string) {
				pdf.Rect(x, y, w, rowH, "D")
				pdf.SetXY(x+1, y+1.5)
				pdf.CellFormat(w-2, 4, text, "", 0, align, false, 0, "")
				x += w
			}
			draw(colN, fmt.Sprintf("%d", row.Number), "C")
			draw(colDate, date, "C")
			draw(colBuyer, buyer, "L")
			draw(colTotal, formatMoney(row.Total), "R")
			draw(colPaid, formatMoney(row.Paid), "R")
			draw(colRem, formatMoney(row.Remaining), "R")
			pdf.SetY(y + rowH)

			sumTotal += row.Total
			sumPaid += row.Paid
			sumRem += row.Remaining
		}

		if pdf.GetY()+rowH > 200 {
			pdf.AddPage()
			writeHeader(firm)
		}
		pdf.SetFont("arial", "B", 8)
		pdf.SetFillColor(245, 245, 245)
		y := pdf.GetY()
		x := left
		drawSum := func(w float64, text, align string) {
			pdf.Rect(x, y, w, rowH, "FD")
			pdf.SetXY(x+1, y+1.5)
			pdf.CellFormat(w-2, 4, text, "", 0, align, false, 0, "")
			x += w
		}
		drawSum(colN+colDate+colBuyer, "Итого", "R")
		drawSum(colTotal, formatMoney(sumTotal), "R")
		drawSum(colPaid, formatMoney(sumPaid), "R")
		drawSum(colRem, formatMoney(sumRem), "R")
		pdf.SetY(y + rowH)
		pdf.SetFillColor(255, 255, 255)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func formatMoney(v float64) string {
	neg := v < 0
	if neg {
		v = -v
	}
	// kopecks as integer to avoid float noise
	kopecks := int64(v*100 + 0.5)
	rubles := kopecks / 100
	kop := kopecks % 100
	s := formatIntGrouped(rubles) + fmt.Sprintf(",%02d", kop)
	if neg {
		return "-" + s
	}
	return s
}

func formatIntGrouped(n int64) string {
	if n < 0 {
		n = -n
	}
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var b []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b = append(b, ' ')
		}
		b = append(b, byte(c))
	}
	return string(b)
}

func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}
