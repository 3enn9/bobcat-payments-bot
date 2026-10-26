package accountant

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"PaymentsBot/internal/db"
	"PaymentsBot/internal/invoice"
	"PaymentsBot/internal/mail"
	"PaymentsBot/internal/max"
)

type Notifier interface {
	SendFileToGroup(groupName, fileName string, reader io.Reader) error
	SendMessageWithPhotos(groupName, text string, photos []max.PhotoUpload) error
	SendMessageInGroupName(nameGroup, message string) error
}

type TelegramInvoiceNotifier interface {
	SendInvoicePages(filename, caption string, pages [][]byte) error
}

type Importer struct {
	DB            *db.Database
	IMAP          mail.IMAPConfig
	Max           Notifier
	Telegram      TelegramInvoiceNotifier
	BackfillSince time.Time // если задано — один тихий прогон с этой даты при старте
}

func (im *Importer) Run(ctx context.Context) {
	if im == nil || im.DB == nil || !im.IMAP.Enabled() {
		return
	}

	log.Println("accountant imap importer started")

	if !im.BackfillSince.IsZero() {
		if err := im.backfillQuiet(im.BackfillSince); err != nil {
			log.Printf("accountant backfill: %v", err)
		}
	}

	im.tick(ctx)
	ticker := time.NewTicker(45 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("accountant imap importer stopped")
			return
		case <-ticker.C:
			im.tick(ctx)
		}
	}
}

func (im *Importer) tick(ctx context.Context) {
	if err := ctx.Err(); err != nil {
		return
	}
	if err := im.poll(); err != nil {
		log.Printf("accountant imap: %v", err)
	}
}

func (im *Importer) poll() error {
	mailbox := im.mailbox()

	lastUID, err := im.DB.GetInvoiceMailUID(mailbox)
	if err != nil {
		return fmt.Errorf("cursor: %w", err)
	}

	messages, maxUID, err := mail.FetchNewPDF(im.IMAP, lastUID)
	if err != nil {
		return err
	}

	for _, msg := range messages {
		for _, att := range msg.Attachments {
			if err := im.importAttachment(att, true, false); err != nil {
				log.Printf("accountant uid=%d file=%s: %v", msg.UID, att.Filename, err)
			}
		}
	}

	if maxUID > lastUID {
		if err := im.DB.SetInvoiceMailUID(mailbox, maxUID); err != nil {
			return fmt.Errorf("save cursor %d: %w", maxUID, err)
		}
	}
	return nil
}

// backfillQuiet тянет счета с почты с даты since без уведомлений в группу.
// После прогона ставит курсор на конец ящика, чтобы обычный poll не повторил всё.
func (im *Importer) backfillQuiet(since time.Time) error {
	mailbox := im.mailbox()
	log.Printf("accountant backfill quiet since %s ...", since.Format("2006-01-02"))

	messages, maxUID, err := mail.FetchPDFSince(im.IMAP, since)
	if err != nil {
		return err
	}

	var imported, updated, skipped, failed, acts int
	for _, msg := range messages {
		for _, att := range msg.Attachments {
			name := strings.TrimSpace(att.Filename)
			if invoice.IsReconciliationFilename(name) {
				acts++
				continue
			}
			if !invoice.IsInvoiceFilename(name) {
				continue
			}
			// replace=true: последний файл по номеру счёта побеждает (в т.ч. без «изм» в имени)
			err := im.importAttachment(att, false, true)
			switch {
			case err == nil:
				imported++
			case errors.Is(err, errInvoiceUpdated):
				updated++
			case errors.Is(err, db.ErrInvoiceExists):
				skipped++
			default:
				failed++
				log.Printf("accountant backfill uid=%d file=%s: %v", msg.UID, name, err)
			}
		}
	}

	if maxUID > 0 {
		if err := im.DB.SetInvoiceMailUID(mailbox, maxUID); err != nil {
			return fmt.Errorf("save cursor %d: %w", maxUID, err)
		}
	}

	log.Printf(
		"accountant backfill done: messages=%d imported=%d updated=%d skipped=%d failed=%d acts_skipped=%d cursor_uid=%d",
		len(messages), imported, updated, skipped, failed, acts, maxUID,
	)
	log.Println("accountant backfill: уберите IMAP_BACKFILL_SINCE из .env после успешного прогона")
	return nil
}

func (im *Importer) mailbox() string {
	if im.IMAP.Mailbox == "" {
		return "INBOX"
	}
	return im.IMAP.Mailbox
}

var errInvoiceUpdated = errors.New("invoice_updated")

// importAttachment: notify — слать в MAX; replace — перезаписывать существующий счёт.
func (im *Importer) importAttachment(att mail.Attachment, notify, replace bool) error {
	name := strings.TrimSpace(att.Filename)
	if invoice.IsReconciliationFilename(name) {
		if notify {
			caption := name
			text, err := invoice.ExtractPDFText(att.Bytes)
			if err != nil {
				log.Printf("accountant reconciliation text: %v", err)
				caption += "\nОткрытые счета: сумма не определена"
			} else if innA, innB, ok := invoice.ReconciliationINNs(text); !ok {
				caption += "\nОткрытые счета: не удалось определить ИНН сторон"
			} else if im.DB == nil {
				caption += "\nОткрытые счета: сумму не удалось рассчитать"
			} else if balance, err := im.DB.OpenInvoiceBalanceBetween(innA, innB); err != nil {
				log.Printf("accountant reconciliation open balance %s/%s: %v", innA, innB, err)
				caption += "\nОткрытые счета: сумму не удалось рассчитать"
			} else {
				caption += fmt.Sprintf("\nСумма открытых неоплаченных счетов: %s ₽", formatReconciliationAmount(balance))
			}

			pages, err := invoice.PDFToImages(att.Bytes, 150)
			if err != nil {
				log.Printf("accountant reconciliation pdf->png: %v", err)
				im.notifyText(caption)
				if im.Max != nil {
					if sendErr := im.Max.SendFileToGroup("Invoices", name, bytes.NewReader(att.Bytes)); sendErr != nil {
						log.Printf("accountant reconciliation max file fallback: %v", sendErr)
					}
				}
				im.sendTelegramPages(name, caption, att.Bytes)
				return nil
			}

			if im.Max != nil {
				photos := make([]max.PhotoUpload, 0, len(pages))
				for i, page := range pages {
					photos = append(photos, max.PhotoUpload{
						Name:   fmt.Sprintf("reconciliation_%d.png", i+1),
						Reader: bytes.NewReader(page),
					})
				}
				if sendErr := im.Max.SendMessageWithPhotos("Invoices", caption, photos); sendErr != nil {
					log.Printf("accountant reconciliation max photos: %v", sendErr)
				}
			}
			im.sendTelegramPageBytes(name, caption, pages)
		}
		return nil
	}
	if !invoice.IsInvoiceFilename(name) {
		return nil
	}

	data, err := invoice.ParsePDF(att.Bytes)
	if err != nil {
		if notify {
			im.notifyText(fmt.Sprintf("Не удалось разобрать %s: %v", name, err))
		}
		return err
	}

	replace = replace || invoice.IsRevisedFilename(name)
	created, err := im.DB.CreateInvoice(toInput(data, replace))
	if errors.Is(err, db.ErrInvoiceExists) {
		log.Printf("accountant skip existing invoice number=%d date=%s", data.Number, data.Date.Format("2006-01-02"))
		return db.ErrInvoiceExists
	}
	if err != nil {
		return err
	}

	if notify {
		caption := fmt.Sprintf("%s\n%s", name, data.BuyerName)
		pages, err := invoice.PDFToImages(att.Bytes, 150)
		if err != nil {
			log.Printf("accountant pdf->png: %v", err)
			if im.Max != nil {
				if sendErr := im.Max.SendFileToGroup("Invoices", name, bytes.NewReader(att.Bytes)); sendErr != nil {
					log.Printf("accountant max file fallback: %v", sendErr)
				}
				im.notifyText(caption)
			}
			im.sendTelegramPages(name, caption, att.Bytes)
		} else {
			if im.Max != nil {
				photos := make([]max.PhotoUpload, 0, len(pages))
				for i, p := range pages {
					photos = append(photos, max.PhotoUpload{Name: fmt.Sprintf("page_%d.png", i+1), Reader: bytes.NewReader(p)})
				}
				if sendErr := im.Max.SendMessageWithPhotos("Invoices", caption, photos); sendErr != nil {
					log.Printf("accountant max photos: %v", sendErr)
				}
			}
			im.sendTelegramPageBytes(name, caption, pages)
		}
	}
	if created != nil && created.Replaced {
		log.Printf("accountant updated invoice number=%d", data.Number)
		return errInvoiceUpdated
	}
	log.Printf("accountant imported invoice number=%d", data.Number)
	return nil
}

func (im *Importer) sendTelegramPages(filename, caption string, pdf []byte) {
	if im == nil || im.Telegram == nil {
		return
	}
	pages, err := invoice.PDFToImages(pdf, 150)
	if err != nil {
		log.Printf("accountant telegram pdf->png: %v", err)
		return
	}
	im.sendTelegramPageBytes(filename, caption, pages)
}

func (im *Importer) sendTelegramPageBytes(filename, caption string, pages [][]byte) {
	if im == nil || im.Telegram == nil {
		return
	}
	if err := im.Telegram.SendInvoicePages(filename, caption, pages); err != nil {
		log.Printf("accountant telegram photos: %v", err)
	}
}

func (im *Importer) notifyText(text string) {
	if im == nil || im.Max == nil || strings.TrimSpace(text) == "" {
		return
	}
	if err := im.Max.SendMessageInGroupName("Invoices", text); err != nil {
		log.Printf("accountant max text: %v", err)
	}
}

func formatReconciliationAmount(amount float64) string {
	return strings.ReplaceAll(fmt.Sprintf("%.2f", amount), ".", ",")
}

func toInput(data *invoice.PDFData, replace bool) db.CreateInvoiceInput {
	items := make([]db.InvoiceItemInput, 0, len(data.Items))
	for _, item := range data.Items {
		items = append(items, db.InvoiceItemInput{
			Position: item.Position,
			Title:    item.Title,
			Quantity: item.Quantity,
			Unit:     item.Unit,
			Price:    item.Price,
			Amount:   item.Amount,
		})
	}
	return db.CreateInvoiceInput{
		Number:          data.Number,
		ReplaceExisting: replace,
		InvoiceDate:     data.Date,
		Basis:           data.Basis,
		SupplierName:    data.SupplierName,
		SupplierINN:     data.SupplierINN,
		SupplierKPP:     data.SupplierKPP,
		SupplierAddress: data.SupplierAddress,
		BankName:        data.BankName,
		BankBIK:         data.BankBIK,
		BankAccount:     data.BankAccount,
		BankCorrAccount: data.BankCorrAccount,
		BuyerName:       data.BuyerName,
		BuyerINN:        data.BuyerINN,
		BuyerKPP:        data.BuyerKPP,
		BuyerAddress:    data.BuyerAddress,
		Total:           data.Total,
		VatAmount:       data.VatAmount,
		Items:           items,
	}
}
