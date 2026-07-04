package banks

import (
	"PaymentsBot/internal/domain/payment"
	"fmt"
	"log"
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

	err := b.messenger.SendMessageInGroupName("Payments", message)
	if err != nil {
		return err
	}

	log.Println("tochkabank send message")

	return nil
}
