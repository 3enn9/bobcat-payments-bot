package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type InvoiceBank struct {
	ID          int64  `json:"id"`
	SupplierID  int64  `json:"supplierId"`
	Name        string `json:"name"`
	BIK         string `json:"bik"`
	Account     string `json:"account"`
	CorrAccount string `json:"corrAccount"`
}

type InvoiceSupplier struct {
	ID                int64  `json:"id"`
	Name              string `json:"name"`
	INN               string `json:"inn"`
	KPP               string `json:"kpp"`
	AddressText       string `json:"addressText"`
	LastInvoiceNumber int    `json:"lastInvoiceNumber"`
}

type InvoiceBuyer struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	INN         string `json:"inn"`
	KPP         string `json:"kpp"`
	AddressText string `json:"addressText"`
	Email       string `json:"email"`
}

type InvoiceItemInput struct {
	Position int     `json:"position"`
	Title    string  `json:"title"`
	Quantity float64 `json:"quantity"`
	Unit     string  `json:"unit"`
	Price    float64 `json:"price"`
	Amount   float64 `json:"amount"`
}

type CreateInvoiceInput struct {
	SupplierID      *int64
	BuyerID         *int64
	BankID          *int64
	Number          int // 0 = next after last_invoice_number
	InvoiceDate     time.Time
	Basis           string
	SupplierName    string
	SupplierINN     string
	SupplierKPP     string
	SupplierAddress string
	BankName        string
	BankBIK         string
	BankAccount     string
	BankCorrAccount string
	BuyerName       string
	BuyerINN        string
	BuyerKPP        string
	BuyerAddress    string
	BuyerEmail      string
	Total           float64
	VatAmount       float64
	Items           []InvoiceItemInput
}

type CreatedInvoice struct {
	ID     int64
	Number int
}

func (d *Database) SearchInvoiceBanks(q string, supplierID int64, limit int) ([]InvoiceBank, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	if supplierID <= 0 {
		return []InvoiceBank{}, nil
	}

	q = strings.TrimSpace(q)
	rows, err := d.DB.Query(`
		SELECT id, supplier_id, name, bik, account, corr_account
		FROM invoice_banks
		WHERE supplier_id = ?
		  AND (? = '' OR name LIKE CONCAT('%', ?, '%') OR bik LIKE CONCAT('%', ?, '%') OR account LIKE CONCAT('%', ?, '%'))
		ORDER BY name
		LIMIT ?
	`, supplierID, q, q, q, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]InvoiceBank, 0)
	for rows.Next() {
		var item InvoiceBank
		if err := rows.Scan(&item.ID, &item.SupplierID, &item.Name, &item.BIK, &item.Account, &item.CorrAccount); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (d *Database) SearchInvoiceSuppliers(q string, limit int) ([]InvoiceSupplier, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	q = strings.TrimSpace(q)
	rows, err := d.DB.Query(`
		SELECT id, name, inn, kpp, address_text, last_invoice_number
		FROM invoice_suppliers
		WHERE (? = '' OR name LIKE CONCAT('%', ?, '%') OR inn LIKE CONCAT('%', ?, '%'))
		ORDER BY name
		LIMIT ?
	`, q, q, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]InvoiceSupplier, 0)
	for rows.Next() {
		var item InvoiceSupplier
		if err := rows.Scan(&item.ID, &item.Name, &item.INN, &item.KPP, &item.AddressText, &item.LastInvoiceNumber); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (d *Database) SearchInvoiceBuyers(q string, limit int) ([]InvoiceBuyer, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	q = strings.TrimSpace(q)
	rows, err := d.DB.Query(`
		SELECT id, name, inn, kpp, address_text, email
		FROM invoice_buyers
		WHERE (? = '' OR name LIKE CONCAT('%', ?, '%') OR inn LIKE CONCAT('%', ?, '%') OR email LIKE CONCAT('%', ?, '%'))
		ORDER BY name
		LIMIT ?
	`, q, q, q, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]InvoiceBuyer, 0)
	for rows.Next() {
		var item InvoiceBuyer
		if err := rows.Scan(&item.ID, &item.Name, &item.INN, &item.KPP, &item.AddressText, &item.Email); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (d *Database) CreateInvoice(input CreateInvoiceInput) (*CreatedInvoice, error) {
	tx, err := d.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	supplierID := input.SupplierID
	if supplierID == nil || *supplierID == 0 {
		result, err := tx.Exec(`
			INSERT INTO invoice_suppliers (name, inn, kpp, address_text, last_invoice_number)
			VALUES (?, ?, ?, ?, 0)
		`, input.SupplierName, input.SupplierINN, input.SupplierKPP, input.SupplierAddress)
		if err != nil {
			return nil, err
		}
		id, err := result.LastInsertId()
		if err != nil {
			return nil, err
		}
		supplierID = &id
	} else {
		_, err := tx.Exec(`
			UPDATE invoice_suppliers
			SET name = ?, inn = ?, kpp = ?, address_text = ?
			WHERE id = ?
		`, input.SupplierName, input.SupplierINN, input.SupplierKPP, input.SupplierAddress, *supplierID)
		if err != nil {
			return nil, err
		}
	}

	bankID := input.BankID
	if bankID != nil && *bankID > 0 {
		var ownerID int64
		err := tx.QueryRow(`
			SELECT supplier_id
			FROM invoice_banks
			WHERE id = ?
		`, *bankID).Scan(&ownerID)
		if err == sql.ErrNoRows || ownerID != *supplierID {
			bankID = nil
		} else if err != nil {
			return nil, err
		}
	}

	if bankID == nil || *bankID == 0 {
		result, err := tx.Exec(`
			INSERT INTO invoice_banks (supplier_id, name, bik, account, corr_account)
			VALUES (?, ?, ?, ?, ?)
		`, *supplierID, input.BankName, input.BankBIK, input.BankAccount, input.BankCorrAccount)
		if err != nil {
			return nil, err
		}
		id, err := result.LastInsertId()
		if err != nil {
			return nil, err
		}
		bankID = &id
	} else {
		_, err := tx.Exec(`
			UPDATE invoice_banks
			SET name = ?, bik = ?, account = ?, corr_account = ?
			WHERE id = ? AND supplier_id = ?
		`, input.BankName, input.BankBIK, input.BankAccount, input.BankCorrAccount, *bankID, *supplierID)
		if err != nil {
			return nil, err
		}
	}

	buyerID := input.BuyerID
	if buyerID == nil || *buyerID == 0 {
		result, err := tx.Exec(`
			INSERT INTO invoice_buyers (name, inn, kpp, address_text, email)
			VALUES (?, ?, ?, ?, ?)
		`, input.BuyerName, input.BuyerINN, input.BuyerKPP, input.BuyerAddress, input.BuyerEmail)
		if err != nil {
			return nil, err
		}
		id, err := result.LastInsertId()
		if err != nil {
			return nil, err
		}
		buyerID = &id
	} else {
		_, err := tx.Exec(`
			UPDATE invoice_buyers
			SET name = ?, inn = ?, kpp = ?, address_text = ?, email = ?
			WHERE id = ?
		`, input.BuyerName, input.BuyerINN, input.BuyerKPP, input.BuyerAddress, input.BuyerEmail, *buyerID)
		if err != nil {
			return nil, err
		}
	}

	var lastNumber int
	if err := tx.QueryRow(`
		SELECT last_invoice_number
		FROM invoice_suppliers
		WHERE id = ?
		FOR UPDATE
	`, *supplierID).Scan(&lastNumber); err != nil {
		return nil, err
	}
	invoiceNumber := input.Number
	if invoiceNumber <= 0 {
		invoiceNumber = lastNumber + 1
	}

	result, err := tx.Exec(`
		INSERT INTO invoices (
			number, invoice_date, basis, supplier_id, buyer_id,
			supplier_name, supplier_inn, supplier_kpp, supplier_address,
			bank_name, bank_bik, bank_account, bank_corr_account,
			buyer_name, buyer_inn, buyer_kpp, buyer_address,
			total, vat_amount
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		invoiceNumber,
		input.InvoiceDate.Format("2006-01-02"),
		input.Basis,
		supplierID,
		buyerID,
		input.SupplierName,
		input.SupplierINN,
		input.SupplierKPP,
		input.SupplierAddress,
		input.BankName,
		input.BankBIK,
		input.BankAccount,
		input.BankCorrAccount,
		input.BuyerName,
		input.BuyerINN,
		input.BuyerKPP,
		input.BuyerAddress,
		input.Total,
		input.VatAmount,
	)
	if err != nil {
		return nil, err
	}

	invoiceID, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	for i, item := range input.Items {
		position := item.Position
		if position <= 0 {
			position = i + 1
		}
		_, err := tx.Exec(`
			INSERT INTO invoice_items (invoice_id, position, title, quantity, unit, price, amount)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, invoiceID, position, item.Title, item.Quantity, item.Unit, item.Price, item.Amount)
		if err != nil {
			return nil, err
		}
	}

	newLast := lastNumber
	if invoiceNumber > newLast {
		newLast = invoiceNumber
	}
	_, err = tx.Exec(`
		UPDATE invoice_suppliers
		SET last_invoice_number = ?
		WHERE id = ?
	`, newLast, *supplierID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &CreatedInvoice{ID: invoiceID, Number: invoiceNumber}, nil
}

func FormatMoney(value float64) string {
	return strings.Replace(fmt.Sprintf("%.2f", value), ".", ",", 1)
}
