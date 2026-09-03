package banks

import (
	"PaymentsBot/internal/db"
	domainPayment "PaymentsBot/internal/domain/payment"
	"errors"
	"fmt"
	"github.com/patrickmn/go-cache"
	"log"
	"strconv"
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

	executedAt, _ := time.Parse(time.RFC3339, payment.DrawDate)
	if executedAt.IsZero() {
		executedAt, _ = time.Parse("2006-01-02", payment.DrawDate)
	}
	if executedAt.IsZero() {
		executedAt = time.Now()
	}
	amount, _ := strconv.ParseFloat(payment.OperationAmount, 64)
	_, saveErr := b.db.SaveIncomingPayment(db.SaveIncomingPaymentInput{
		Source:        db.SourceTbank,
		ExternalID:    payment.OperationID,
		ExecutedAt:    executedAt,
		Amount:        amount,
		Account:       payment.AccountNumber,
		RecipientName: payment.Receiver.Name,
		PayerName:     payment.CounterParty.Name,
		PayerINN:      payment.CounterParty.Inn,
		Purpose:       payment.Description,
		RawDocNumber:  payment.DocumentNumber,
	})
	if saveErr != nil && !errors.Is(saveErr, db.ErrIncomingPaymentExists) {
		log.Printf("tbank save payment: %v", saveErr)
	}

	err := b.messenger.SendMessageInGroupName("Payments", message)
	if err != nil {
		return err
	}

	log.Printf("TBank %+v", payment)

	return nil
}
