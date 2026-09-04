package db

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

var (
	ErrPaymentAlreadyMatched = errors.New("payment_already_matched")
	ErrInvoiceAlreadyPaid    = errors.New("invoice_already_paid")
	ErrMatchAmountMismatch   = errors.New("match_amount_mismatch")
	ErrMatchEmpty            = errors.New("match_empty")
	ErrMatchForeignInvoice   = errors.New("match_foreign_invoice")
)

type MatchFirm struct {
	ID                 int64  `json:"id"`
	Name               string `json:"name"`
	INN                string `json:"inn"`
	UnmatchedCount     int    `json:"unmatchedCount"`
	UnpaidInvoiceCount int    `json:"unpaidInvoiceCount"`
}

type MatchPaymentItem struct {
	ID         int64     `json:"id"`
	Source     string    `json:"source"`
	ExecutedAt time.Time `json:"executedAt"`
	Amount     float64   `json:"amount"`
	PayerName  string    `json:"payerName"`
	PayerINN   string    `json:"payerInn"`
	Purpose    string    `json:"purpose"`
	Account    string    `json:"account"`
}

type MatchInvoiceItem struct {
	ID          int64     `json:"id"`
	Number      int       `json:"number"`
	InvoiceDate time.Time `json:"invoiceDate"`
	BuyerName   string    `json:"buyerName"`
	BuyerINN    string    `json:"buyerInn"`
	Total       float64   `json:"total"`
}

// ListMatchFirms — поставщики с нераспределёнными входящими платежами на их р/с.
func (d *Database) ListMatchFirms() ([]MatchFirm, error) {
	rows, err := d.DB.Query(`
		SELECT s.id, s.name, s.inn,
		       COUNT(DISTINCT p.id) AS unmatched_count,
		       (
		         SELECT COUNT(*)
		         FROM invoices i
		         WHERE i.supplier_id = s.id AND i.incoming_payment_id IS NULL
		       ) AS unpaid_count
		FROM invoice_suppliers s
		INNER JOIN invoice_banks b ON b.supplier_id = s.id
		INNER JOIN incoming_payments p ON p.account = b.account
		WHERE NOT EXISTS (
		  SELECT 1 FROM invoices i WHERE i.incoming_payment_id = p.id
		)
		GROUP BY s.id, s.name, s.inn
		HAVING unmatched_count > 0
		ORDER BY s.name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]MatchFirm, 0)
	for rows.Next() {
		var f MatchFirm
		if err := rows.Scan(&f.ID, &f.Name, &f.INN, &f.UnmatchedCount, &f.UnpaidInvoiceCount); err != nil {
			return nil, err
		}
		result = append(result, f)
	}
	return result, rows.Err()
}

// ListUnmatchedPaymentsForSupplier — входящие на р/с фирмы, ещё никуда не назначенные.
func (d *Database) ListUnmatchedPaymentsForSupplier(supplierID int64) ([]MatchPaymentItem, error) {
	rows, err := d.DB.Query(`
		SELECT p.id, p.source, p.executed_at, p.amount,
		       p.payer_name, p.payer_inn, p.purpose, p.account
		FROM incoming_payments p
		INNER JOIN invoice_banks b ON b.account = p.account AND b.supplier_id = ?
		WHERE NOT EXISTS (
		  SELECT 1 FROM invoices i WHERE i.incoming_payment_id = p.id
		)
		ORDER BY p.executed_at DESC, p.id DESC
	`, supplierID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]MatchPaymentItem, 0)
	for rows.Next() {
		var item MatchPaymentItem
		if err := rows.Scan(
			&item.ID, &item.Source, &item.ExecutedAt, &item.Amount,
			&item.PayerName, &item.PayerINN, &item.Purpose, &item.Account,
		); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

// ListUnpaidInvoicesForSupplier — счета фирмы без привязанного платежа.
func (d *Database) ListUnpaidInvoicesForSupplier(supplierID int64) ([]MatchInvoiceItem, error) {
	rows, err := d.DB.Query(`
		SELECT id, number, invoice_date, buyer_name, buyer_inn, total
		FROM invoices
		WHERE supplier_id = ? AND incoming_payment_id IS NULL
		ORDER BY invoice_date DESC, number DESC
	`, supplierID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]MatchInvoiceItem, 0)
	for rows.Next() {
		var item MatchInvoiceItem
		if err := rows.Scan(
			&item.ID, &item.Number, &item.InvoiceDate, &item.BuyerName, &item.BuyerINN, &item.Total,
		); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

// MatchPaymentToInvoices привязывает один платёж к нескольким счетам.
// Суммы должны совпасть до копейки.
func (d *Database) MatchPaymentToInvoices(paymentID int64, invoiceIDs []int64) error {
	if paymentID <= 0 || len(invoiceIDs) == 0 {
		return ErrMatchEmpty
	}

	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var paymentAmount float64
	var paymentAccount string
	err = tx.QueryRow(`
		SELECT amount, account
		FROM incoming_payments
		WHERE id = ?
		FOR UPDATE
	`, paymentID).Scan(&paymentAmount, &paymentAccount)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("платёж не найден")
	}
	if err != nil {
		return err
	}

	var alreadyMatched int
	if err := tx.QueryRow(`
		SELECT COUNT(*) FROM invoices WHERE incoming_payment_id = ?
	`, paymentID).Scan(&alreadyMatched); err != nil {
		return err
	}
	if alreadyMatched > 0 {
		return ErrPaymentAlreadyMatched
	}

	var supplierID int64
	err = tx.QueryRow(`
		SELECT supplier_id FROM invoice_banks WHERE account = ? LIMIT 1
	`, paymentAccount).Scan(&supplierID)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("р/с платежа не привязан к поставщику")
	}
	if err != nil {
		return err
	}

	placeholders := make([]string, len(invoiceIDs))
	idArgs := make([]any, len(invoiceIDs))
	for i, id := range invoiceIDs {
		placeholders[i] = "?"
		idArgs[i] = id
	}

	rows, err := tx.Query(fmt.Sprintf(`
		SELECT id, total, incoming_payment_id, supplier_id
		FROM invoices
		WHERE id IN (%s)
		FOR UPDATE
	`, strings.Join(placeholders, ",")), idArgs...)
	if err != nil {
		return err
	}

	found := make(map[int64]float64, len(invoiceIDs))
	var sum float64
	for rows.Next() {
		var id int64
		var invSupplier sql.NullInt64
		var total float64
		var payment sql.NullInt64
		if err := rows.Scan(&id, &total, &payment, &invSupplier); err != nil {
			rows.Close()
			return err
		}
		if !invSupplier.Valid || invSupplier.Int64 != supplierID {
			rows.Close()
			return ErrMatchForeignInvoice
		}
		if payment.Valid {
			rows.Close()
			return ErrInvoiceAlreadyPaid
		}
		found[id] = total
		sum += total
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	if len(found) != len(invoiceIDs) {
		return fmt.Errorf("часть счетов не найдена")
	}

	sum = math.Round(sum*100) / 100
	paymentAmount = math.Round(paymentAmount*100) / 100
	if sum != paymentAmount {
		return ErrMatchAmountMismatch
	}

	for _, id := range invoiceIDs {
		res, err := tx.Exec(`
			UPDATE invoices SET incoming_payment_id = ? WHERE id = ? AND incoming_payment_id IS NULL
		`, paymentID, id)
		if err != nil {
			return err
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return ErrInvoiceAlreadyPaid
		}
	}

	return tx.Commit()
}
