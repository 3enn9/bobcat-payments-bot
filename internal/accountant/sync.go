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

type Importer struct {
	DB            *db.Database
	IMAP          mail.IMAPConfig
	Max           Notifier
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
			if err := im.importAttachment(att, true); err != nil {
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

	var imported, skipped, failed, acts int
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
			err := im.importAttachment(att, false)
			switch {
			case err == nil:
				imported++
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
		"accountant backfill done: messages=%d imported=%d skipped=%d failed=%d acts_skipped=%d cursor_uid=%d",
		len(messages), imported, skipped, failed, acts, maxUID,
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

// importAttachment: notify=true — шлёт в MAX; notify=false — только БД.
// ErrInvoiceExists возвращается наружу (для статистики бэкофилла).
func (im *Importer) importAttachment(att mail.Attachment, notify bool) error {
	name := strings.TrimSpace(att.Filename)
	if invoice.IsReconciliationFilename(name) {
		if notify {
			im.notifyText("Акт сверки: " + name)
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

	_, err = im.DB.CreateInvoice(toInput(data, invoice.IsRevisedFilename(name)))
	if errors.Is(err, db.ErrInvoiceExists) {
		log.Printf("accountant skip existing invoice %s №%d", data.SupplierName, data.Number)
		return db.ErrInvoiceExists
	}
	if err != nil {
		return err
	}

	if notify && im.Max != nil {
		caption := fmt.Sprintf("%s\n%s", name, data.BuyerName)
		pages, err := invoice.PDFToImages(att.Bytes, 150)
		if err != nil {
			log.Printf("accountant pdf->png: %v", err)
			if sendErr := im.Max.SendFileToGroup("Invoices", name, bytes.NewReader(att.Bytes)); sendErr != nil {
				log.Printf("accountant max file fallback: %v", sendErr)
			}
			im.notifyText(caption)
		} else {
			photos := make([]max.PhotoUpload, 0, len(pages))
			for i, p := range pages {
				photos = append(photos, max.PhotoUpload{
					Name:   fmt.Sprintf("page_%d.png", i+1),
					Reader: bytes.NewReader(p),
				})
			}
			if sendErr := im.Max.SendMessageWithPhotos("Invoices", caption, photos); sendErr != nil {
				log.Printf("accountant max photos: %v", sendErr)
			}
		}
	}
	log.Printf("accountant imported %s №%d %s", data.SupplierName, data.Number, data.BuyerName)
	return nil
}

func (im *Importer) notifyText(text string) {
	if im == nil || im.Max == nil || strings.TrimSpace(text) == "" {
		return
	}
	if err := im.Max.SendMessageInGroupName("Invoices", text); err != nil {
		log.Printf("accountant max text: %v", err)
	}
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
