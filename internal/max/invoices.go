package max

import (
	"PaymentsBot/internal/clock"
	"PaymentsBot/internal/invoice"
	"bytes"
	"fmt"
	"log"
	"sort"
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
				Number:            item.Number,
				Date:              item.InvoiceDate,
				BuyerName:         item.BuyerName,
				IsRegularCustomer: item.IsRegularCustomer,
				Total:             item.Total,
				Paid:              item.PaidAmount,
				Remaining:         item.RemainingAmount,
			})
		}
		code := strings.ToUpper(invoice.SupplierFileCode(firm.Name, firm.INN))
		if strings.TrimSpace(firm.INN) == "645499004617" {
			regular, other := make([]invoice.UnpaidTableRow, 0), make([]invoice.UnpaidTableRow, 0)
			for _, row := range rows {
				if row.IsRegularCustomer {
					regular = append(regular, row)
				} else {
					other = append(other, row)
				}
			}
			sort.SliceStable(regular, func(i, j int) bool {
				return normalizedBuyerName(regular[i].BuyerName) < normalizedBuyerName(regular[j].BuyerName)
			})
			sort.SliceStable(other, func(i, j int) bool {
				return other[i].Number < other[j].Number
			})
			if len(regular) > 0 && m.sendUnpaidInvoiceTable(chatID, code, firm.Name, firm.INN, "Постоянщики", regular, now) {
				sent++
			}
			if len(other) > 0 && m.sendUnpaidInvoiceTable(chatID, code, firm.Name, firm.INN, "Остальные выставленные счета", other, now) {
				sent++
			}
			continue
		}
		if m.sendUnpaidInvoiceTable(chatID, code, firm.Name, firm.INN, "", rows, now) {
			sent++
		}
	}

	if sent == 0 {
		_ = m.SendMessageInGroupID(chatID, formatFirmUnpaidCaption("", "", 0, now))
	}
}

func (m *MaxService) sendUnpaidInvoiceTable(chatID int64, code, firmName, firmINN, section string, rows []invoice.UnpaidTableRow, now time.Time) bool {
	table := invoice.UnpaidFirmTable{
		FirmName:    firmName,
		FirmINN:     firmINN,
		SectionName: section,
		Rows:        rows,
	}
	pdfBytes, err := invoice.GenerateUnpaidTablesPDF([]invoice.UnpaidFirmTable{table}, now)
	if err != nil {
		log.Printf("invoices cmd: pdf firm=%s section=%s: %v", firmName, section, err)
		return false
	}

	captionCode := code
	fileCode := strings.ToLower(code)
	if section != "" {
		captionCode += " · " + section
		fileCode += "_" + section
	}
	caption := formatFirmUnpaidCaption(captionCode, firmName, len(rows), now)
	pages, err := invoice.PDFToImages(pdfBytes, 150)
	if err != nil {
		log.Printf("invoices cmd: pdf->png firm=%s section=%s: %v", firmName, section, err)
		fileName := fmt.Sprintf("unpaid_%s.pdf", sanitizeFilePart(fileCode))
		if sendErr := m.SendFileToChat(chatID, fileName, bytes.NewReader(pdfBytes)); sendErr != nil {
			log.Printf("invoices cmd: pdf fallback firm=%s section=%s: %v", firmName, section, sendErr)
			return false
		}
		_ = m.SendMessageInGroupID(chatID, caption)
		return true
	}

	photos := make([]PhotoUpload, 0, len(pages))
	for i, page := range pages {
		photos = append(photos, PhotoUpload{
			Name:   fmt.Sprintf("%s_%d.png", sanitizeFilePart(fileCode), i+1),
			Reader: bytes.NewReader(page),
		})
	}
	if err := m.SendPhotosToChat(chatID, caption, photos); err != nil {
		log.Printf("invoices cmd: send photos firm=%s section=%s: %v", firmName, section, err)
		_ = m.SendMessageInGroupID(chatID, fmt.Sprintf("Не удалось отправить %s", captionCode))
		return false
	}
	return true
}

func normalizedBuyerName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	return strings.ReplaceAll(name, "ё", "е")
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
