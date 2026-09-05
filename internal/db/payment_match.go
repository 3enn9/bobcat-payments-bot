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
	ErrPaymentFullyMatched = errors.New("payment_fully_matched")
	ErrInvoiceAlreadyPaid  = errors.New("invoice_already_paid")
	ErrMatchEmpty          = errors.New("match_empty")
	ErrMatchForeignInvoice = errors.New("match_foreign_invoice")
	ErrMatchWrongPayer     = errors.New("match_wrong_payer")
)

type MatchFirm struct {
	ID                 int64  `json:"id"`
	Name               string `json:"name"`
	INN                string `json:"inn"`
	UnmatchedCount     int    `json:"unmatchedCount"`
	UnpaidInvoiceCount int    `json:"unpaidInvoiceCount"`
}

type MatchPaymentItem struct {
	ID              int64     `json:"id"`
	Source          string    `json:"source"`
	ExecutedAt      time.Time `json:"executedAt"`
	Amount          float64   `json:"amount"`
	RemainingAmount float64   `json:"remainingAmount"`
	PayerName       string    `json:"payerName"`
	PayerINN        string    `json:"payerInn"`
	Purpose         string    `json:"purpose"`
	Account         string    `json:"account"`
}

type MatchInvoiceItem struct {
	ID              int64     `json:"id"`
	Number          int       `json:"number"`
	InvoiceDate     time.Time `json:"invoiceDate"`
	BuyerName       string    `json:"buyerName"`
	BuyerINN        string    `json:"buyerInn"`
	Total           float64   `json:"total"`
	PaidAmount      float64   `json:"paidAmount"`
	RemainingAmount float64   `json:"remainingAmount"`
}

func roundMoney(v float64) float64 {
	return math.Round(v*100) / 100
}

// ListMatchFirms — поставщики с платежами, у которых ещё есть нераспределённый остаток.
func (d *Database) ListMatchFirms() ([]MatchFirm, error) {
	rows, err := d.DB.Query(`
		SELECT s.id, s.name, s.inn,
		       COUNT(DISTINCT p.id) AS unmatched_count,
		       (
		         SELECT COUNT(*)
		         FROM invoices i
		         WHERE i.supplier_id = s.id
		           AND i.status = 'open'
		           AND ROUND(i.total - IFNULL((
		             SELECT SUM(a.amount) FROM invoice_payment_allocations a WHERE a.invoice_id = i.id
		           ), 0), 2) > 0
		       ) AS unpaid_count
		FROM invoice_suppliers s
		INNER JOIN invoice_banks b ON b.supplier_id = s.id
		INNER JOIN incoming_payments p ON p.account = b.account
		WHERE ROUND(p.amount - IFNULL((
		  SELECT SUM(a.amount) FROM invoice_payment_allocations a WHERE a.payment_id = p.id
		), 0), 2) > 0
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

// ListUnmatchedPaymentsForSupplier — платежи с остатком > 0 на р/с фирмы.
func (d *Database) ListUnmatchedPaymentsForSupplier(supplierID int64) ([]MatchPaymentItem, error) {
	rows, err := d.DB.Query(`
		SELECT p.id, p.source, p.executed_at, p.amount,
		       ROUND(p.amount - IFNULL(SUM(a.amount), 0), 2) AS remaining,
		       p.payer_name, p.payer_inn, p.purpose, p.account
		FROM incoming_payments p
		INNER JOIN invoice_banks b ON b.account = p.account AND b.supplier_id = ?
		LEFT JOIN invoice_payment_allocations a ON a.payment_id = p.id
		GROUP BY p.id, p.source, p.executed_at, p.amount, p.payer_name, p.payer_inn, p.purpose, p.account
		HAVING remaining > 0
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
			&item.ID, &item.Source, &item.ExecutedAt, &item.Amount, &item.RemainingAmount,
			&item.PayerName, &item.PayerINN, &item.Purpose, &item.Account,
		); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

// ListOpenInvoicesForSupplier — счета с остатком к оплате; если payerINN задан — только этого покупателя.
func (d *Database) ListOpenInvoicesForSupplier(supplierID int64, payerINN, payerName string) ([]MatchInvoiceItem, error) {
	query := `
		SELECT i.id, i.number, i.invoice_date, i.buyer_name, i.buyer_inn, i.total,
		       ROUND(IFNULL(SUM(a.amount), 0), 2) AS paid,
		       ROUND(i.total - IFNULL(SUM(a.amount), 0), 2) AS remaining,
		       i.status
		FROM invoices i
		LEFT JOIN invoice_payment_allocations a ON a.invoice_id = i.id
		WHERE i.supplier_id = ?
		  AND i.status = 'open'
	`
	args := []any{supplierID}

	payerINN = strings.TrimSpace(payerINN)
	payerName = strings.TrimSpace(payerName)
	if payerINN != "" {
		query += ` AND i.buyer_inn = ?`
		args = append(args, payerINN)
	} else if payerName != "" {
		query += ` AND LOWER(REPLACE(i.buyer_name, 'ё', 'е')) = LOWER(REPLACE(?, 'ё', 'е'))`
		args = append(args, payerName)
	}

	query += `
		GROUP BY i.id, i.number, i.invoice_date, i.buyer_name, i.buyer_inn, i.total, i.status
		HAVING remaining > 0
		ORDER BY i.invoice_date ASC, i.number ASC
	`

	rows, err := d.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]MatchInvoiceItem, 0)
	for rows.Next() {
		var item MatchInvoiceItem
		var status string
		if err := rows.Scan(
			&item.ID, &item.Number, &item.InvoiceDate, &item.BuyerName, &item.BuyerINN, &item.Total,
			&item.PaidAmount, &item.RemainingAmount, &status,
		); err != nil {
			return nil, err
		}
		if status != "open" {
			continue
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

// MatchPaymentToInvoices распределяет остаток платежа по выбранным счетам по порядку.
// Частичная оплата допускается: остаток платежа и остаток счёта сохраняются.
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
	var paymentAccount, payerINN, payerName string
	err = tx.QueryRow(`
		SELECT amount, account, payer_inn, payer_name
		FROM incoming_payments
		WHERE id = ?
		FOR UPDATE
	`, paymentID).Scan(&paymentAmount, &paymentAccount, &payerINN, &payerName)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("платёж не найден")
	}
	if err != nil {
		return err
	}

	var allocated float64
	if err := tx.QueryRow(`
		SELECT ROUND(IFNULL(SUM(amount), 0), 2) FROM invoice_payment_allocations WHERE payment_id = ?
	`, paymentID).Scan(&allocated); err != nil {
		return err
	}
	remainingPayment := roundMoney(paymentAmount - allocated)
	if remainingPayment <= 0 {
		return ErrPaymentFullyMatched
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
		SELECT i.id, i.total, i.supplier_id, i.buyer_inn, i.buyer_name, i.status,
		       ROUND(IFNULL((
		         SELECT SUM(a.amount) FROM invoice_payment_allocations a WHERE a.invoice_id = i.id
		       ), 0), 2) AS paid
		FROM invoices i
		WHERE i.id IN (%s)
		FOR UPDATE
	`, strings.Join(placeholders, ",")), idArgs...)
	if err != nil {
		return err
	}

	type invRow struct {
		id        int64
		remaining float64
	}
	byID := make(map[int64]invRow, len(invoiceIDs))
	for rows.Next() {
		var id int64
		var total float64
		var invSupplier sql.NullInt64
		var buyerINN, buyerName, status string
		var paid float64
		if err := rows.Scan(&id, &total, &invSupplier, &buyerINN, &buyerName, &status, &paid); err != nil {
			_ = rows.Close()
			return err
		}
		if status != "open" {
			_ = rows.Close()
			return ErrInvoiceAlreadyPaid
		}
		if !invSupplier.Valid || invSupplier.Int64 != supplierID {
			_ = rows.Close()
			return ErrMatchForeignInvoice
		}
		if !payerMatches(payerINN, payerName, buyerINN, buyerName) {
			_ = rows.Close()
			return ErrMatchWrongPayer
		}
		rem := roundMoney(total - paid)
		if rem <= 0 {
			_ = rows.Close()
			return ErrInvoiceAlreadyPaid
		}
		byID[id] = invRow{id: id, remaining: rem}
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}

	if len(byID) != len(invoiceIDs) {
		return fmt.Errorf("часть счетов не найдена")
	}

	for _, id := range invoiceIDs {
		if remainingPayment <= 0 {
			break
		}
		inv := byID[id]
		alloc := inv.remaining
		if alloc > remainingPayment {
			alloc = remainingPayment
		}
		alloc = roundMoney(alloc)
		if alloc <= 0 {
			continue
		}

		if _, err := tx.Exec(`
			INSERT INTO invoice_payment_allocations (payment_id, invoice_id, amount)
			VALUES (?, ?, ?)
			ON DUPLICATE KEY UPDATE amount = ROUND(amount + VALUES(amount), 2)
		`, paymentID, id, alloc); err != nil {
			return err
		}
		remainingPayment = roundMoney(remainingPayment - alloc)

		if roundMoney(inv.remaining-alloc) <= 0 {
			if _, err := tx.Exec(`UPDATE invoices SET status = 'paid' WHERE id = ?`, id); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func payerMatches(payerINN, payerName, buyerINN, buyerName string) bool {
	payerINN = strings.TrimSpace(payerINN)
	buyerINN = strings.TrimSpace(buyerINN)
	if payerINN != "" && buyerINN != "" {
		return payerINN == buyerINN
	}
	pn := normalizePartyName(payerName)
	bn := normalizePartyName(buyerName)
	if pn == "" || bn == "" {
		return false
	}
	return pn == bn
}

func normalizePartyName(s string) string {
	s = strings.ToLower(strings.TrimSpace(strings.ReplaceAll(s, "\u00a0", " ")))
	s = strings.ReplaceAll(s, "ё", "е")
	s = strings.Join(strings.Fields(s), " ")
	return s
}
