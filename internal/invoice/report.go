package invoice

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
)

type BunkerHaulReportMessage struct {
	SentAt         string
	Text           string
	AnnotationKind string
	Amount         float64
}

func GenerateBunkerHaulReportPDF(surname string, date time.Time, messages []BunkerHaulReportMessage, comment string) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(14, 14, 14)
	pdf.SetAutoPageBreak(true, 14)
	pdf.AddUTF8FontFromBytes("arial", "", fontRegular)
	pdf.AddUTF8FontFromBytes("arial", "B", fontBold)
	pdf.AddPage()

	pdf.SetFont("arial", "B", 16)
	pdf.MultiCell(0, 8, fmt.Sprintf("Отчёт по вывозам: %s", surname), "", "L", false)
	pdf.SetFont("arial", "", 11)
	pdf.CellFormat(0, 7, date.Format("02.01.2006"), "", 1, "L", false, 0, "")
	pdf.Ln(5)

	for i, message := range messages {
		pdf.SetFont("arial", "B", 11)
		pdf.MultiCell(0, 6, fmt.Sprintf("%d. %s", i+1, message.SentAt), "", "L", false)
		pdf.SetFont("arial", "", 11)
		pdf.MultiCell(0, 5.5, message.Text, "", "L", false)
		if label := bunkerAnnotationLabel(message.AnnotationKind, message.Amount); label != "" {
			pdf.SetFont("arial", "B", 10)
			pdf.MultiCell(0, 5.5, "Отметка: "+label, "", "L", false)
		}
		pdf.SetDrawColor(220, 220, 220)
		y := pdf.GetY() + 2
		pdf.Line(14, y, 196, y)
		pdf.SetY(y + 4)
	}

	pdf.Ln(2)
	pdf.SetFont("arial", "B", 12)
	pdf.CellFormat(0, 7, "Комментарий", "", 1, "L", false, 0, "")
	pdf.SetFont("arial", "", 11)
	if strings.TrimSpace(comment) == "" {
		comment = "—"
	}
	pdf.MultiCell(0, 5.5, comment, "", "L", false)

	var output bytes.Buffer
	if err := pdf.Output(&output); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func bunkerAnnotationLabel(kind string, amount float64) string {
	if kind == "" || amount <= 0 {
		return ""
	}
	value := strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.3f", amount), "0"), ".")
	switch kind {
	case "vector":
		return "Вектор — вес: " + value
	case "dubki":
		return "Дубки — сумма: " + value + " ₽"
	case "cash":
		return "Нал — сумма: " + value + " ₽"
	default:
		return ""
	}
}
