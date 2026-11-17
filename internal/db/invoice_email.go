package db

import (
	"database/sql"
	"errors"
	"time"
)

var (
	ErrInvoiceBuyerNotFound      = errors.New("invoice buyer not found")
	ErrInvoiceBuyerEmailExists   = errors.New("invoice buyer email already exists")
	ErrInvoiceBuyerEmailNotFound = errors.New("invoice buyer email not found")
	ErrInvoiceEmailNotOpen       = errors.New("invoice is not open")
	ErrInvoiceNoRecipientEmails  = errors.New("invoice buyer has no email addresses")
)

type InvoiceEmailFirm struct {
	ID                 int64  `json:"id"`
	Name               string `json:"name"`
	INN                string `json:"inn"`
	UnpaidInvoiceCount int    `json:"unpaidInvoiceCount"`
}

type InvoiceBuyerEmail struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
}

type InvoiceEmailData struct {
	Number          int
	Date            time.Time
	Basis           string
	BankName        string
	BankBIK         string
	BankAccount     string
	BankCorrAccount string
	SupplierName    string
	SupplierINN     string
	SupplierKPP     string
	SupplierAddress string
	BuyerID         int64
	BuyerName       string
	BuyerINN        string
	BuyerKPP        string
	BuyerAddress    string
	Emails          []InvoiceBuyerEmail
	Items           []InvoiceItemInput
	Total           float64
	VatAmount       float64
}

func (d *Database) ListUnpaidInvoiceEmailFirms() ([]InvoiceEmailFirm, error) {
	rows, err := d.DB.Query(`
		SELECT b.id, b.name, b.inn, COUNT(*) AS unpaid_invoice_count
		FROM invoices i
		INNER JOIN invoice_buyers b ON b.id = i.buyer_id
		WHERE i.status = 'open'
		  AND ROUND(i.total - IFNULL((
		    SELECT SUM(a.amount)
		    FROM invoice_payment_allocations a
		    WHERE a.invoice_id = i.id
		  ), 0), 2) > 0
		GROUP BY b.id, b.name, b.inn
		ORDER BY b.name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	firms := make([]InvoiceEmailFirm, 0)
	for rows.Next() {
		var firm InvoiceEmailFirm
		if err := rows.Scan(&firm.ID, &firm.Name, &firm.INN, &firm.UnpaidInvoiceCount); err != nil {
			return nil, err
		}
		firms = append(firms, firm)
	}
	return firms, rows.Err()
}

func (d *Database) ListInvoiceBuyerEmails(buyerID int64) ([]InvoiceBuyerEmail, error) {
	rows, err := d.DB.Query(`
		SELECT e.id, e.email
		FROM invoice_buyer_emails e
		WHERE e.buyer_id = ?
		ORDER BY e.email
	`, buyerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	emails := make([]InvoiceBuyerEmail, 0)
	for rows.Next() {
		var email InvoiceBuyerEmail
		if err := rows.Scan(&email.ID, &email.Email); err != nil {
			return nil, err
		}
		emails = append(emails, email)
	}
	return emails, rows.Err()
}

func (d *Database) AddInvoiceBuyerEmail(buyerID int64, email string) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var exists bool
	if err := tx.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM invoice_buyers b
			INNER JOIN invoices i ON i.buyer_id = b.id
			WHERE b.id = ?
			  AND i.status = 'open'
			  AND ROUND(i.total - IFNULL((
			    SELECT SUM(a.amount) FROM invoice_payment_allocations a WHERE a.invoice_id = i.id
			  ), 0), 2) > 0
		)
	`, buyerID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrInvoiceBuyerNotFound
	}

	result, err := tx.Exec(`
		INSERT IGNORE INTO invoice_buyer_emails (buyer_id, email)
		VALUES (?, ?)
	`, buyerID, email)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrInvoiceBuyerEmailExists
	}

	if _, err = tx.Exec(`
		UPDATE invoice_buyers
		SET email = ?
		WHERE id = ? AND email = ''
	`, email, buyerID); err != nil {
		return err
	}
	return tx.Commit()
}

func (d *Database) DeleteInvoiceBuyerEmail(buyerID, emailID int64) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var email string
	err = tx.QueryRow(`
		SELECT email FROM invoice_buyer_emails WHERE id = ? AND buyer_id = ?
	`, emailID, buyerID).Scan(&email)
	if err == sql.ErrNoRows {
		return ErrInvoiceBuyerEmailNotFound
	}
	if err != nil {
		return err
	}

	result, err := tx.Exec(`
		DELETE FROM invoice_buyer_emails WHERE id = ? AND buyer_id = ?
	`, emailID, buyerID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrInvoiceBuyerEmailNotFound
	}

	if _, err = tx.Exec(`
		UPDATE invoice_buyers
		SET email = COALESCE((
			SELECT e.email FROM invoice_buyer_emails e
			WHERE e.buyer_id = ? ORDER BY e.id LIMIT 1
		), '')
		WHERE id = ? AND email = ?
	`, buyerID, buyerID, email); err != nil {
		return err
	}
	return tx.Commit()
}

func (d *Database) LoadOpenInvoiceEmailData(invoiceID int64) (*InvoiceEmailData, error) {
	var data InvoiceEmailData
	var buyerID sql.NullInt64
	var status string
	var remaining float64
	err := d.DB.QueryRow(`
		SELECT i.number, i.invoice_date, i.basis,
		       i.bank_name, i.bank_bik, i.bank_account, i.bank_corr_account,
		       i.supplier_name, i.supplier_inn, i.supplier_kpp, i.supplier_address,
		       i.buyer_id, i.buyer_name, i.buyer_inn, i.buyer_kpp, i.buyer_address,
		       i.total, i.vat_amount, i.status,
		       ROUND(i.total - IFNULL((
		         SELECT SUM(a.amount) FROM invoice_payment_allocations a WHERE a.invoice_id = i.id
		       ), 0), 2) AS remaining
		FROM invoices i
		WHERE i.id = ?
	`, invoiceID).Scan(
		&data.Number, &data.Date, &data.Basis,
		&data.BankName, &data.BankBIK, &data.BankAccount, &data.BankCorrAccount,
		&data.SupplierName, &data.SupplierINN, &data.SupplierKPP, &data.SupplierAddress,
		&buyerID, &data.BuyerName, &data.BuyerINN, &data.BuyerKPP, &data.BuyerAddress,
		&data.Total, &data.VatAmount, &status, &remaining,
	)
	if err == sql.ErrNoRows {
		return nil, ErrInvoiceEmailNotOpen
	}
	if err != nil {
		return nil, err
	}
	if status != "open" || remaining <= 0 {
		return nil, ErrInvoiceEmailNotOpen
	}
	if !buyerID.Valid || buyerID.Int64 <= 0 {
		return nil, ErrInvoiceNoRecipientEmails
	}
	data.BuyerID = buyerID.Int64

	data.Emails, err = d.ListInvoiceBuyerEmails(data.BuyerID)
	if err != nil {
		return nil, err
	}
	if len(data.Emails) == 0 {
		return nil, ErrInvoiceNoRecipientEmails
	}

	rows, err := d.DB.Query(`
		SELECT position, title, quantity, unit, price, amount
		FROM invoice_items
		WHERE invoice_id = ?
		ORDER BY position, id
	`, invoiceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	data.Items = make([]InvoiceItemInput, 0)
	for rows.Next() {
		var item InvoiceItemInput
		if err := rows.Scan(&item.Position, &item.Title, &item.Quantity, &item.Unit, &item.Price, &item.Amount); err != nil {
			return nil, err
		}
		data.Items = append(data.Items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &data, nil
}
