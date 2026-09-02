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
)

type Notifier interface {
	SendFileToGroup(groupName, fileName string, reader io.Reader) error
	SendMessageInGroupName(nameGroup, message string) error
}

type Importer struct {
	DB   *db.Database
	IMAP mail.IMAPConfig
	Max  Notifier
}

func (im *Importer) Run(ctx context.Context) {
	if im == nil || im.DB == nil || !im.IMAP.Enabled() {
		return
	}

	log.Println("accountant imap importer started")
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
	mailbox := im.IMAP.Mailbox
	if mailbox == "" {
		mailbox = "INBOX"
	}

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
			if err := im.importAttachment(att); err != nil {
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

func (im *Importer) importAttachment(att mail.Attachment) error {
	name := strings.TrimSpace(att.Filename)
	if invoice.IsReconciliationFilename(name) {
		im.notifyText("Акт сверки: " + name)
		return nil
	}
	if !invoice.IsInvoiceFilename(name) {
		return nil
	}

	data, err := invoice.ParsePDF(att.Bytes)
	if err != nil {
		im.notifyText(fmt.Sprintf("Не удалось разобрать %s: %v", name, err))
		return err
	}

	_, err = im.DB.CreateInvoice(toInput(data, invoice.IsRevisedFilename(name)))
	if errors.Is(err, db.ErrInvoiceExists) {
		log.Printf("accountant skip existing invoice %s №%d", data.SupplierName, data.Number)
		return nil
	}
	if err != nil {
		return err
	}

	caption := fmt.Sprintf(
		"%s\n%s\n%s",
		name,
		data.BuyerName,
		formatAmount(data.Total),
	)
	im.notifyText(caption)
	if im.Max != nil {
		if sendErr := im.Max.SendFileToGroup("Invoices", name, bytes.NewReader(att.Bytes)); sendErr != nil {
			log.Printf("accountant max file: %v", sendErr)
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

func formatAmount(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	return strings.Replace(s, ".", ",", 1)
}
