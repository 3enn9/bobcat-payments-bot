package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type InvoiceBank struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	BIK         string `json:"bik"`
	Account     string `json:"account"`
	CorrAccount string `json:"corrAccount"`
}

type InvoiceSupplier struct {
	ID                int64         `json:"id"`
	Name              string        `json:"name"`
	INN               string        `json:"inn"`
	KPP               string        `json:"kpp"`
	AddressText       string        `json:"addressText"`
	BankID            *int64        `json:"bankId"`
	LastInvoiceNumber int           `json:"lastInvoiceNumber"`
	Bank              *InvoiceBank  `json:"bank,omitempty"`
}

type InvoiceBuyer struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	INN         string `json:"inn"`
	KPP         string `json:"kpp"`
	AddressText string `json:"addressText"`
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
	Total           float64
	VatAmount       float64
	Items           []InvoiceItemInput
}

type CreatedInvoice struct {
	ID     int64
	Number int
}

func (d *Database) SearchInvoiceBanks(q string, limit int) ([]InvoiceBank, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	q = strings.TrimSpace(q)
	rows, err := d.DB.Query(`
		SELECT id, name, bik, account, corr_account
		FROM invoice_banks
		WHERE (? = '' OR name LIKE CONCAT('%', ?, '%') OR bik LIKE CONCAT('%', ?, '%'))
		ORDER BY name
		LIMIT ?
	`, q, q, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]InvoiceBank, 0)
	for rows.Next() {
		var item InvoiceBank
		if err := rows.Scan(&item.ID, &item.Name, &item.BIK, &item.Account, &item.CorrAccount); err != nil {
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
		SELECT s.id, s.name, s.inn, s.kpp, s.address_text, s.bank_id, s.last_invoice_number,
		       b.id, b.name, b.bik, b.account, b.corr_account
		FROM invoice_suppliers s
		LEFT JOIN invoice_banks b ON b.id = s.bank_id
		WHERE (? = '' OR s.name LIKE CONCAT('%', ?, '%') OR s.inn LIKE CONCAT('%', ?, '%'))
		ORDER BY s.name
		LIMIT ?
	`, q, q, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]InvoiceSupplier, 0)
	for rows.Next() {
		var item InvoiceSupplier
		var bankID sql.NullInt64
		var bID sql.NullInt64
		var bName, bBIK, bAccount, bCorr sql.NullString
		if err := rows.Scan(
			&item.ID, &item.Name, &item.INN, &item.KPP, &item.AddressText, &bankID, &item.LastInvoiceNumber,
			&bID, &bName, &bBIK, &bAccount, &bCorr,
		); err != nil {
			return nil, err
		}
		if bankID.Valid {
			id := bankID.Int64
			item.BankID = &id
		}
		if bID.Valid {
			item.Bank = &InvoiceBank{
				ID:          bID.Int64,
				Name:        bName.String,
				BIK:         bBIK.String,
				Account:     bAccount.String,
				CorrAccount: bCorr.String,
			}
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
		SELECT id, name, inn, kpp, address_text
		FROM invoice_buyers
		WHERE (? = '' OR name LIKE CONCAT('%', ?, '%') OR inn LIKE CONCAT('%', ?, '%'))
		ORDER BY name
		LIMIT ?
	`, q, q, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]InvoiceBuyer, 0)
	for rows.Next() {
		var item InvoiceBuyer
		if err := rows.Scan(&item.ID, &item.Name, &item.INN, &item.KPP, &item.AddressText); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (d *Database) UpsertInvoiceBank(id *int64, name, bik, account, corrAccount string) (int64, error) {
	name = strings.TrimSpace(name)
	bik = strings.TrimSpace(bik)
	account = strings.TrimSpace(account)
	corrAccount = strings.TrimSpace(corrAccount)

	if id != nil && *id > 0 {
		_, err := d.DB.Exec(`
			UPDATE invoice_banks
			SET name = ?, bik = ?, account = ?, corr_account = ?
			WHERE id = ?
		`, name, bik, account, corrAccount, *id)
		if err != nil {
			return 0, err
		}
		return *id, nil
	}

	result, err := d.DB.Exec(`
		INSERT INTO invoice_banks (name, bik, account, corr_account)
		VALUES (?, ?, ?, ?)
	`, name, bik, account, corrAccount)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (d *Database) UpsertInvoiceSupplier(
	id *int64,
	name, inn, kpp, addressText string,
	bankID *int64,
) (int64, error) {
	name = strings.TrimSpace(name)
	inn = strings.TrimSpace(inn)
	kpp = strings.TrimSpace(kpp)
	addressText = strings.TrimSpace(addressText)

	if id != nil && *id > 0 {
		_, err := d.DB.Exec(`
			UPDATE invoice_suppliers
			SET name = ?, inn = ?, kpp = ?, address_text = ?, bank_id = ?
			WHERE id = ?
		`, name, inn, kpp, addressText, bankID, *id)
		if err != nil {
			return 0, err
		}
		return *id, nil
	}

	result, err := d.DB.Exec(`
		INSERT INTO invoice_suppliers (name, inn, kpp, address_text, bank_id, last_invoice_number)
		VALUES (?, ?, ?, ?, ?, 0)
	`, name, inn, kpp, addressText, bankID)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (d *Database) UpsertInvoiceBuyer(id *int64, name, inn, kpp, addressText string) (int64, error) {
	name = strings.TrimSpace(name)
	inn = strings.TrimSpace(inn)
	kpp = strings.TrimSpace(kpp)
	addressText = strings.TrimSpace(addressText)

	if id != nil && *id > 0 {
		_, err := d.DB.Exec(`
			UPDATE invoice_buyers
			SET name = ?, inn = ?, kpp = ?, address_text = ?
			WHERE id = ?
		`, name, inn, kpp, addressText, *id)
		if err != nil {
			return 0, err
		}
		return *id, nil
	}

	result, err := d.DB.Exec(`
		INSERT INTO invoice_buyers (name, inn, kpp, address_text)
		VALUES (?, ?, ?, ?)
	`, name, inn, kpp, addressText)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (d *Database) CreateInvoice(input CreateInvoiceInput) (*CreatedInvoice, error) {
	tx, err := d.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	bankID := input.BankID
	if bankID == nil || *bankID == 0 {
		result, err := tx.Exec(`
			INSERT INTO invoice_banks (name, bik, account, corr_account)
			VALUES (?, ?, ?, ?)
		`, input.BankName, input.BankBIK, input.BankAccount, input.BankCorrAccount)
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
			WHERE id = ?
		`, input.BankName, input.BankBIK, input.BankAccount, input.BankCorrAccount, *bankID)
		if err != nil {
			return nil, err
		}
	}

	supplierID := input.SupplierID
	if supplierID == nil || *supplierID == 0 {
		result, err := tx.Exec(`
			INSERT INTO invoice_suppliers (name, inn, kpp, address_text, bank_id, last_invoice_number)
			VALUES (?, ?, ?, ?, ?, 0)
		`, input.SupplierName, input.SupplierINN, input.SupplierKPP, input.SupplierAddress, bankID)
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
			SET name = ?, inn = ?, kpp = ?, address_text = ?, bank_id = ?
			WHERE id = ?
		`, input.SupplierName, input.SupplierINN, input.SupplierKPP, input.SupplierAddress, bankID, *supplierID)
		if err != nil {
			return nil, err
		}
	}

	buyerID := input.BuyerID
	if buyerID == nil || *buyerID == 0 {
		result, err := tx.Exec(`
			INSERT INTO invoice_buyers (name, inn, kpp, address_text)
			VALUES (?, ?, ?, ?)
		`, input.BuyerName, input.BuyerINN, input.BuyerKPP, input.BuyerAddress)
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
			SET name = ?, inn = ?, kpp = ?, address_text = ?
			WHERE id = ?
		`, input.BuyerName, input.BuyerINN, input.BuyerKPP, input.BuyerAddress, *buyerID)
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
	nextNumber := lastNumber + 1

	result, err := tx.Exec(`
		INSERT INTO invoices (
			number, invoice_date, basis, supplier_id, buyer_id,
			supplier_name, supplier_inn, supplier_kpp, supplier_address,
			bank_name, bank_bik, bank_account, bank_corr_account,
			buyer_name, buyer_inn, buyer_kpp, buyer_address,
			total, vat_amount
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		nextNumber,
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

	_, err = tx.Exec(`
		UPDATE invoice_suppliers
		SET last_invoice_number = ?
		WHERE id = ?
	`, nextNumber, *supplierID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &CreatedInvoice{ID: invoiceID, Number: nextNumber}, nil
}

func FormatMoney(value float64) string {
	return strings.Replace(fmt.Sprintf("%.2f", value), ".", ",", 1)
}
