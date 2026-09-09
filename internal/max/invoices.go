package max

import (
	"PaymentsBot/internal/clock"
	"PaymentsBot/internal/invoice"
	"bytes"
	"fmt"
	"log"
	"time"

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

	tables := make([]invoice.UnpaidFirmTable, 0, len(firms))
	totalInvoices := 0
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
		tables = append(tables, invoice.UnpaidFirmTable{
			FirmName: firm.Name,
			FirmINN:  firm.INN,
			Rows:     rows,
		})
		totalInvoices += len(rows)
	}

	now := clock.Now()
	pdfBytes, err := invoice.GenerateUnpaidTablesPDF(tables, now)
	if err != nil {
		log.Printf("invoices cmd: pdf: %v", err)
		_ = m.SendMessageInGroupID(chatID, "Не удалось сформировать таблицу.")
		return
	}

	caption := formatUnpaidCaption(now, len(tables), totalInvoices)
	pages, err := invoice.PDFToImages(pdfBytes, 150)
	if err != nil {
		log.Printf("invoices cmd: pdf->png: %v", err)
		if sendErr := m.SendFileToChat(chatID, "unpaid_invoices.pdf", bytes.NewReader(pdfBytes)); sendErr != nil {
			log.Printf("invoices cmd: pdf fallback: %v", sendErr)
			_ = m.SendMessageInGroupID(chatID, "Не удалось отправить таблицу.")
			return
		}
		_ = m.SendMessageInGroupID(chatID, caption)
		return
	}

	const batchSize = 8
	for i := 0; i < len(pages); i += batchSize {
		end := i + batchSize
		if end > len(pages) {
			end = len(pages)
		}
		photos := make([]PhotoUpload, 0, end-i)
		for j, p := range pages[i:end] {
			photos = append(photos, PhotoUpload{
				Name:   fmt.Sprintf("invoices_%d.png", i+j+1),
				Reader: bytes.NewReader(p),
			})
		}
		text := ""
		if i == 0 {
			text = caption
		}
		if err := m.SendPhotosToChat(chatID, text, photos); err != nil {
			log.Printf("invoices cmd: send photos: %v", err)
			_ = m.SendMessageInGroupID(chatID, "Не удалось отправить фото таблиц.")
			return
		}
	}
}

func formatUnpaidCaption(at time.Time, firms, invoices int) string {
	when := at.Format("02.01.2006 15:04")
	if firms == 0 {
		return fmt.Sprintf("Неоплаченных счетов нет · %s", when)
	}
	return fmt.Sprintf("Неоплаченные счета · %s\nФирм: %d · Счетов: %d", when, firms, invoices)
}
