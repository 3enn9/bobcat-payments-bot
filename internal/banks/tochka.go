package banks

import (
	"PaymentsBot/internal/db"
	"PaymentsBot/internal/domain/payment"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"
)

func DateFormatTochka(date string) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	} else {
		return t.Format("02.01.2006")
	}
}

func (b *BankService) TochkaBank(payment payment.TochkaPayment) error {

	date := DateFormatTochka(payment.Date)

	message := fmt.Sprintf(
		"🏦 %s\n\n"+
			"👤 Плательщик: %s\n"+
			"🏢 Получатель: %s\n\n"+
			"🧾 Назначение:\n%s\n\n"+
			"💰 Оплата:\n"+
			"<pre>%s %s %s</pre>",
		payment.SideRecipient.BankName,
		payment.SidePayer.Name,
		payment.SideRecipient.Name,
		payment.Purpose,
		date,
		payment.SidePayer.Amount,
		"точка",
	)

	executedAt, _ := time.Parse("2006-01-02", payment.Date)
	if executedAt.IsZero() {
		executedAt = time.Now()
	}
	amount, _ := strconv.ParseFloat(payment.SidePayer.Amount, 64)
	_, saveErr := b.db.SaveIncomingPayment(db.SaveIncomingPaymentInput{
		Source:        db.SourceTochka,
		ExternalID:    payment.PaymentId,
		ExecutedAt:    executedAt,
		Amount:        amount,
		Currency:      payment.SidePayer.Currency,
		Account:       payment.SideRecipient.Account,
		RecipientName: payment.SideRecipient.Name,
		PayerName:     payment.SidePayer.Name,
		PayerINN:      payment.SidePayer.Inn,
		Purpose:       payment.Purpose,
		RawDocNumber:  payment.DocumentNumber,
	})
	if saveErr != nil && !errors.Is(saveErr, db.ErrIncomingPaymentExists) {
		log.Printf("tochka save payment: %v", saveErr)
	}

	err := b.messenger.SendMessageInGroupName("Payments", message)
	if err != nil {
		return err
	}

	log.Println("tochkabank send message")

	return nil
}
