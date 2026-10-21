package db

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"log"
	"net"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

type IncomingPaymentSource string

const (
	SourceModulbank IncomingPaymentSource = "modulbank"
	SourceTochka    IncomingPaymentSource = "tochka"
	SourceTbank     IncomingPaymentSource = "tbank"
)

type IncomingPayment struct {
	ID            int64
	Source        IncomingPaymentSource
	ExternalID    string
	ExecutedAt    time.Time
	Amount        float64
	Currency      string
	Account       string
	RecipientName string
	PayerName     string
	PayerINN      string
	Purpose       string
	RawDocNumber  string
	CreatedAt     time.Time
}

type SaveIncomingPaymentInput struct {
	Source        IncomingPaymentSource
	ExternalID    string
	ExecutedAt    time.Time
	Amount        float64
	Currency      string
	Account       string
	RecipientName string
	PayerName     string
	PayerINN      string
	Purpose       string
	RawDocNumber  string
}

var (
	ErrIncomingPaymentExists    = errors.New("incoming_payment_exists")
	ErrInvalidIncomingPaymentID = errors.New("incoming_payment_id_required")
)

const incomingPaymentSaveAttempts = 3

// SaveIncomingPayment сохраняет входящий платёж.
// Возвращает ErrIncomingPaymentExists если такой (source, external_id) уже есть.
func (d *Database) SaveIncomingPayment(inp SaveIncomingPaymentInput) (int64, error) {
	inp.ExternalID = strings.TrimSpace(inp.ExternalID)
	if inp.ExternalID == "" || len(inp.ExternalID) > 128 {
		return 0, ErrInvalidIncomingPaymentID
	}
	if inp.Currency == "" {
		inp.Currency = "RUB"
	}
	var res sql.Result
	var err error
	for attempt := 1; attempt <= incomingPaymentSaveAttempts; attempt++ {
		res, err = d.DB.Exec(`
			INSERT IGNORE INTO incoming_payments
				(source, external_id, executed_at, amount, currency,
				 account, recipient_name, payer_name, payer_inn, purpose, raw_doc_number, match_status)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'open')
		`,
			inp.Source,
			inp.ExternalID,
			inp.ExecutedAt.Format("2006-01-02"),
			inp.Amount,
			inp.Currency,
			inp.Account,
			inp.RecipientName,
			inp.PayerName,
			inp.PayerINN,
			inp.Purpose,
			inp.RawDocNumber,
		)
		if err == nil {
			break
		}
		if !isRetryableIncomingPaymentError(err) || attempt == incomingPaymentSaveAttempts {
			log.Printf("incoming payment save failed source=%s external_id=%q attempts=%d: %v", inp.Source, inp.ExternalID, attempt, err)
			return 0, err
		}

		log.Printf("incoming payment save retry source=%s external_id=%q attempt=%d/%d: %v", inp.Source, inp.ExternalID, attempt, incomingPaymentSaveAttempts, err)
		time.Sleep(time.Duration(attempt) * 200 * time.Millisecond)
	}
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if id == 0 {
		return 0, ErrIncomingPaymentExists
	}
	return id, nil
}

func isRetryableIncomingPaymentError(err error) bool {
	if errors.Is(err, driver.ErrBadConn) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}

	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		switch mysqlErr.Number {
		case 1205, // lock wait timeout
			1213, // deadlock victim
			2006, // server has gone away
			2013: // lost connection during query
			return true
		}
	}

	var networkErr net.Error
	return errors.As(err, &networkErr)
}

type IncomingPaymentsFilter struct {
	DateFrom time.Time
	DateTo   time.Time
	Source   IncomingPaymentSource
	Account  string
	Limit    int
}

// ListIncomingPayments возвращает входящие платежи с фильтрацией.
func (d *Database) ListIncomingPayments(f IncomingPaymentsFilter) ([]IncomingPayment, error) {
	if f.Limit <= 0 || f.Limit > 500 {
		f.Limit = 100
	}

	query := `
		SELECT id, source, external_id, executed_at, amount, currency,
		       account, recipient_name, payer_name, payer_inn, purpose, raw_doc_number, created_at
		FROM incoming_payments
		WHERE 1=1`

	var args []any

	if !f.DateFrom.IsZero() {
		query += " AND executed_at >= ?"
		args = append(args, f.DateFrom.Format("2006-01-02"))
	}
	if !f.DateTo.IsZero() {
		query += " AND executed_at <= ?"
		args = append(args, f.DateTo.Format("2006-01-02"))
	}
	if f.Source != "" {
		query += " AND source = ?"
		args = append(args, f.Source)
	}
	if f.Account != "" {
		query += " AND account = ?"
		args = append(args, f.Account)
	}

	query += " ORDER BY executed_at DESC, id DESC LIMIT ?"
	args = append(args, f.Limit)

	rows, err := d.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []IncomingPayment
	for rows.Next() {
		var p IncomingPayment
		var executedAt, createdAt string
		if err := rows.Scan(
			&p.ID, &p.Source, &p.ExternalID, &executedAt, &p.Amount, &p.Currency,
			&p.Account, &p.RecipientName, &p.PayerName, &p.PayerINN, &p.Purpose, &p.RawDocNumber, &createdAt,
		); err != nil {
			return nil, err
		}
		p.ExecutedAt, _ = time.Parse("2006-01-02", executedAt)
		p.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		result = append(result, p)
	}
	return result, rows.Err()
}

// GetIncomingPaymentByExternal ищет уже сохранённый платёж.
func (d *Database) GetIncomingPaymentByExternal(source IncomingPaymentSource, externalID string) (*IncomingPayment, error) {
	var p IncomingPayment
	var executedAt, createdAt string
	err := d.DB.QueryRow(`
		SELECT id, source, external_id, executed_at, amount, currency,
		       account, recipient_name, payer_name, payer_inn, purpose, raw_doc_number, created_at
		FROM incoming_payments
		WHERE source = ? AND external_id = ?
		LIMIT 1
	`, source, externalID).Scan(
		&p.ID, &p.Source, &p.ExternalID, &executedAt, &p.Amount, &p.Currency,
		&p.Account, &p.RecipientName, &p.PayerName, &p.PayerINN, &p.Purpose, &p.RawDocNumber, &createdAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.ExecutedAt, _ = time.Parse("2006-01-02", executedAt)
	p.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	return &p, nil
}
