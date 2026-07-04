package banks

import (
	"PaymentsBot/internal/domain/payment"
	"fmt"
	"log"
	"time"
)

func (b *BankService) ModuleBank(payload payment.ModulbankPayment) error {

	if payload.Operation.Category != "Debet" {
		return payment.ErrTypeOfOperation
	}

	recipientName := "Неизвестный получатель"

	switch payload.Operation.BankAccountNumber {
	case "40702810670010185610":
		recipientName = `ООО "СарСтройТех"`
	case "40802810870010171379":
		recipientName = `ИП Архипов Николай Николаевич`
	case "40802810670010198701":
		recipientName = `ИП Архипов Николай Владимирович`
	default:
		log.Println("Операция не на расчетном счете")
	}

	executed := DateFormatModule(payload.Operation.Executed)

	message := fmt.Sprintf(
		"🏦 %s\n\n"+
			"👤 Плательщик: %s\n"+
			"🏢 Получатель: %s\n\n"+
			"🧾 Назначение:\n%s\n\n"+
			"💰 Оплата:\n"+
			"<pre>%s %.0f %s</pre>",
		`АО "Модульбанк"`,
		payload.Operation.ContragentName,
		recipientName,
		payload.Operation.PaymentPurpose,
		executed,
		payload.Operation.Amount,
		"модуль",
	)

	err := b.messenger.SendMessageInGroupName("Payments", message)
	if err != nil {
		return err
	}

	log.Println("modulebank send message")
	return nil
}

func DateFormatModule(date string) string {
	t, err := time.Parse("2006-01-02T15:04:05", date)
	if err != nil {
		return date
	} else {
		return t.Format("02.01.2006")
	}

}
