package db

import "database/sql"

const invoiceMailMailbox = "INBOX"

func (d *Database) GetInvoiceMailUID(mailbox string) (uint32, error) {
	if mailbox == "" {
		mailbox = invoiceMailMailbox
	}
	var uid uint32
	err := d.DB.QueryRow(`
		SELECT last_uid FROM invoice_mail_cursor WHERE mailbox = ?
	`, mailbox).Scan(&uid)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return uid, err
}

func (d *Database) SetInvoiceMailUID(mailbox string, uid uint32) error {
	if mailbox == "" {
		mailbox = invoiceMailMailbox
	}
	_, err := d.DB.Exec(`
		INSERT INTO invoice_mail_cursor (mailbox, last_uid)
		VALUES (?, ?)
		ON DUPLICATE KEY UPDATE last_uid = VALUES(last_uid)
	`, mailbox, uid)
	return err
}
