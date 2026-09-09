package max

import (
	"PaymentsBot/internal/clock"
	"PaymentsBot/internal/invoice"
	"bytes"
	"fmt"
	"log"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/max-messenger/max-bot-api-client-go/schemes"
)

func (m *MaxService) handleInvoicesCommand(upd *schemes.MessageCreatedUpdate) {
	chatID := upd.GetChatID()
	_ = m.SendMessageInGroupID(chatID, "Собираю неоплаченные счета…")

	firms, err := m.db.ListUnpaidInvoiceFirms()
	if err != nil {
		log.Printf("invoices cmd: list firms: %v", err)
		_ = m.SendMessageInGroupID(chatID, "Не удалось загрузить список фирм.")
		return
	}

	now := clock.Now()
	sent := 0
	for _, firm := range firms {
		items, err := m.db.ListOpenInvoicesForSupplier(firm.ID, "", "")
		if err != nil {
			log.Printf("invoices cmd: list invoices supplier=%d: %v", firm.ID, err)
			continue
		}
		if len(items) == 0 {
			continue
		}
		rows := make([]invoice.UnpaidTableRow, 0, len(items))
		for _, item := range items {
			rows = append(rows, invoice.UnpaidTableRow{
				Number:    item.Number,
				Date:      item.InvoiceDate,
				BuyerName: item.BuyerName,
				Total:     item.Total,
				Paid:      item.PaidAmount,
				Remaining: item.RemainingAmount,
			})
		}
		table := invoice.UnpaidFirmTable{
			FirmName: firm.Name,
			FirmINN:  firm.INN,
			Rows:     rows,
		}

		pdfBytes, err := invoice.GenerateUnpaidTablesPDF([]invoice.UnpaidFirmTable{table}, now)
		if err != nil {
			log.Printf("invoices cmd: pdf firm=%s: %v", firm.Name, err)
			continue
		}

		code := strings.ToUpper(invoice.SupplierFileCode(firm.Name, firm.INN))
		caption := formatFirmUnpaidCaption(code, firm.Name, len(rows), now)

		pages, err := invoice.PDFToImages(pdfBytes, 150)
		if err != nil {
			log.Printf("invoices cmd: pdf->png firm=%s: %v", firm.Name, err)
			fileName := fmt.Sprintf("unpaid_%s.pdf", sanitizeFilePart(code))
			if sendErr := m.SendFileToChat(chatID, fileName, bytes.NewReader(pdfBytes)); sendErr != nil {
				log.Printf("invoices cmd: pdf fallback firm=%s: %v", firm.Name, sendErr)
				continue
			}
			_ = m.SendMessageInGroupID(chatID, caption)
			sent++
			continue
		}

		photos := make([]PhotoUpload, 0, len(pages))
		for i, p := range pages {
			photos = append(photos, PhotoUpload{
				Name:   fmt.Sprintf("%s_%d.png", sanitizeFilePart(code), i+1),
				Reader: bytes.NewReader(p),
			})
		}
		if err := m.SendPhotosToChat(chatID, caption, photos); err != nil {
			log.Printf("invoices cmd: send photos firm=%s: %v", firm.Name, err)
			_ = m.SendMessageInGroupID(chatID, fmt.Sprintf("Не удалось отправить %s", code))
			continue
		}
		sent++
	}

	if sent == 0 {
		_ = m.SendMessageInGroupID(chatID, formatFirmUnpaidCaption("", "", 0, now))
	}
}

func formatFirmUnpaidCaption(code, firmName string, invoices int, at time.Time) string {
	when := at.Format("02.01.2006 15:04")
	if invoices == 0 || firmName == "" {
		return fmt.Sprintf("Неоплаченных счетов нет · %s", when)
	}
	if code != "" {
		return fmt.Sprintf("%s\n%s\nСчетов: %d · %s", code, firmName, invoices, when)
	}
	return fmt.Sprintf("%s\nСчетов: %d · %s", firmName, invoices, when)
}

func sanitizeFilePart(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "firm"
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r >= '0' && r <= '9',
			r >= 'А' && r <= 'Я', r >= 'а' && r <= 'я', r == 'Ё' || r == 'ё':
			b.WriteRune(r)
		default:
			if utf8.RuneCountInString(b.String()) > 0 {
				b.WriteByte('_')
			}
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "firm"
	}
	return out
}
