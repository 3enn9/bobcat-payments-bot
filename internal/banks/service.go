package banks

import (
	"PaymentsBot/internal/db"
	domainPayment "PaymentsBot/internal/domain/payment"
	multi "PaymentsBot/internal/multiMessenger"
	"encoding/json"
	"fmt"
	"time"
)

func NewBankService(messenger *multi.MultiMessenger, database *db.Database) *BankService {
	return &BankService{messenger: messenger, db: database}
}

type BankService struct {
	messenger *multi.MultiMessenger
	db        *db.Database
}

func (b *BankService) ProcessWebhook(source string, payload []byte) error {
	switch db.IncomingPaymentSource(source) {
	case db.SourceTochka:
		var payment domainPayment.TochkaPayment
		if err := json.Unmarshal(payload, &payment); err != nil {
			return fmt.Errorf("decode queued Tochka payment: %w", err)
		}
		return b.TochkaBank(payment)
	case db.SourceModulbank:
		var payment domainPayment.ModulbankPayment
		if err := json.Unmarshal(payload, &payment); err != nil {
			return fmt.Errorf("decode queued Modulbank payment: %w", err)
		}
		return b.ModuleBank(payment)
	case db.SourceTbank:
		var payment domainPayment.TBankPayment
		if err := json.Unmarshal(payload, &payment); err != nil {
			return fmt.Errorf("decode queued TBank payment: %w", err)
		}
		return b.TBank(payment)
	default:
		return fmt.Errorf("unknown queued payment source %q", source)
	}
}

func formatRFC3339(date string) string {
	t, err := time.Parse(time.RFC3339, date)
	if err != nil {
		return date
	}
	return t.Format("02.01.2006")
}
