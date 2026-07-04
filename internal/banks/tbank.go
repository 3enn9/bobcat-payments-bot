package banks

import (
	domainPayment "PaymentsBot/internal/domain/payment"
	"fmt"
	"github.com/patrickmn/go-cache"
	"log"
	"time"
)

var tbankCache = cache.New(5*time.Minute, 10*time.Minute)

func (b *BankService) TBank(payment domainPayment.TBankPayment) error {

	if payment.TypeOfOperation == "Debit" {
		return domainPayment.ErrTypeOfOperation
	}

	opID := payment.OperationID

	if _, found := tbankCache.Get(opID); found {
		log.Println("tbank dublicate:", opID)
		return nil
	}
	tbankCache.Set(opID, true, cache.DefaultExpiration)

	date := formatRFC3339(payment.DrawDate)

	message := fmt.Sprintf(
		"🏦 %s\n\n"+
			"👤 Плательщик: %s\n"+
			"🏢 Получатель: %s\n\n"+
			"🧾 Назначение:\n%s\n\n"+
			"💰 Оплата:\n"+
			"<pre>%s %s %s</pre>",
		`АО "ТБанк"`,
		payment.CounterParty.Name,
		payment.Receiver.Name,
		payment.Description,
		date,
		payment.OperationAmount,
		"тбанк",
	)

	err := b.messenger.SendMessageInGroupName("Payments", message)
	if err != nil {
		return err
	}

	log.Printf("TBank %+v", payment)

	return nil
}
