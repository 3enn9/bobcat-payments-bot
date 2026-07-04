package banks

import (
	multi "PaymentsBot/internal/multiMessenger"
	"time"
)

func NewBankService(messenger *multi.MultiMessenger) *BankService {
	return &BankService{messenger: messenger}
}

type BankService struct {
	messenger *multi.MultiMessenger
}

func formatRFC3339(date string) string {
	t, err := time.Parse(time.RFC3339, date)
	if err != nil {
		return date
	}
	return t.Format("02.01.2006")
}
