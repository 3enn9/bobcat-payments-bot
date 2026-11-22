package invoice

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
)

type BunkerHaulReportMessage struct {
	SentAt      string
	Text        string
	Annotations []BunkerHaulReportAnnotation
}

type BunkerHaulReportAnnotation struct {
	Kind   string
	Amount float64
}

type BunkerHaulReportCashOperation struct {
	EntryType   string
	Amount      float64
	Description string
}

func GenerateBunkerHaulReportPDF(
	surname string,
	date time.Time,
	messages []BunkerHaulReportMessage,
	cashOperations []BunkerHaulReportCashOperation,
	comment string,
	cashBalance float64,
) ([]byte, error) {
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
		pdf.Write(5.5, fmt.Sprintf("%d. %s ", i+1, message.SentAt))
		pdf.SetFont("arial", "", 11)
		pdf.Write(5.5, message.Text)
		pdf.Ln(5.5)
		for _, annotation := range message.Annotations {
			label := bunkerAnnotationLabel(annotation.Kind, annotation.Amount)
			if label == "" {
				continue
			}
			pdf.SetFont("arial", "B", 10)
			pdf.MultiCell(0, 5.5, label, "", "L", false)
		}
		pdf.SetDrawColor(220, 220, 220)
		y := pdf.GetY() + 2
		pdf.Line(14, y, 196, y)
		pdf.SetY(y + 4)
	}

	if len(cashOperations) > 0 {
		pdf.Ln(2)
		pdf.SetFont("arial", "B", 12)
		pdf.CellFormat(0, 7, "Операции кассы", "", 1, "L", false, 0, "")
		for i, operation := range cashOperations {
			kind := "Приход"
			if operation.EntryType == "expense" {
				kind = "Расход"
			}
			line := fmt.Sprintf("%d. %s — %s ₽", i+1, kind, formatPDFMoney(operation.Amount))
			if description := strings.TrimSpace(operation.Description); description != "" {
				line += " — " + description
			}
			pdf.SetFont("arial", "", 11)
			pdf.MultiCell(0, 5.5, line, "", "L", false)
		}
	}

	pdf.Ln(2)
	pdf.SetFont("arial", "B", 12)
	pdf.CellFormat(0, 7, "Комментарий", "", 1, "L", false, 0, "")
	pdf.SetFont("arial", "", 11)
	if strings.TrimSpace(comment) == "" {
		comment = "—"
	}
	pdf.MultiCell(0, 5.5, comment, "", "L", false)
	pdf.Ln(3)
	pdf.SetFont("arial", "B", 12)
	pdf.CellFormat(0, 7, "Касса: "+formatPDFMoney(cashBalance)+" ₽", "", 1, "L", false, 0, "")

	var output bytes.Buffer
	if err := pdf.Output(&output); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func bunkerAnnotationLabel(kind string, amount float64) string {
	if kind == "base" {
		return "База"
	}
	if kind == "rsm" {
		return "РСМ"
	}
	if kind == "" || amount <= 0 {
		return ""
	}
	value := strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.3f", amount), "0"), ".")
	switch kind {
	case "vector":
		return "Вектор — вес: " + value
	case "dubki":
		return "Дубки — " + value + " ₽"
	case "cash":
		return "Нал — " + value + " ₽"
	default:
		return ""
	}
}
