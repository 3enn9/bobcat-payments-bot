package banks

import (
	"PaymentsBot/internal/db"
	multi "PaymentsBot/internal/multiMessenger"
	"time"
)

func NewBankService(messenger *multi.MultiMessenger, database *db.Database) *BankService {
	return &BankService{messenger: messenger, db: database}
}

type BankService struct {
	messenger *multi.MultiMessenger
	db        *db.Database
}

func formatRFC3339(date string) string {
	t, err := time.Parse(time.RFC3339, date)
	if err != nil {
		return date
	}
	return t.Format("02.01.2006")
}
