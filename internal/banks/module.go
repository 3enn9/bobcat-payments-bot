package banks

import (
	"PaymentsBot/internal/tg"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type Operation struct {
	ID        string `json:"id"`
	CompanyID string `json:"companyId"`
	Status    string `json:"status"`
	Category  string `json:"category"`

	ContragentName              string `json:"contragentName"`
	ContragentInn               string `json:"contragentInn"`
	ContragentKpp               string `json:"contragentKpp"`
	ContragentBankAccountNumber string `json:"contragentBankAccountNumber"`
	ContragentBankName          string `json:"contragentBankName"`
	ContragentBankBic           string `json:"contragentBankBic"`

	Currency          string  `json:"currency"`
	Amount            float64 `json:"amount"`
	BankAccountNumber string  `json:"bankAccountNumber"`
	PaymentPurpose    string  `json:"paymentPurpose"`

	Executed string `json:"executed"`
	Created  string `json:"created"`

	DocNumber    string `json:"docNumber"`
	Kbk          string `json:"kbk"`
	Oktmo        string `json:"oktmo"`
	PaymentBasis string `json:"paymentBasis"`

	TaxCode     string `json:"taxCode"`
	TaxDocNum   string `json:"taxDocNum"`
	TaxDocDate  string `json:"taxDocDate"`
	PayerStatus string `json:"payerStatus"`
	Uin         string `json:"uin"`

	AbsID  string `json:"absId"`
	IbsoID string `json:"ibsoId"`
	CardID string `json:"cardId"`
}

type ModulbankWebhook struct {
	CompanyInn string    `json:"companyInn"`
	CompanyKpp string    `json:"companyKpp"`
	Operation  Operation `json:"operation"`
	SHA1Hash   string    `json:"SHA1Hash"`
}

func ModuleBankHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}

	var payload ModulbankWebhook
	err = json.Unmarshal(bodyBytes, &payload)
	if err != nil {
		http.Error(w, "error marshaling", http.StatusBadRequest)
		return
	}

	if payload.Operation.Category != "Debet" {
		w.WriteHeader(http.StatusOK)
		log.Println("Не входящий платеж")
		return
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
		w.WriteHeader(http.StatusOK)
		log.Println("Операция не на расчетном счете")
		return

	}

	message := fmt.Sprintf(
		"🏦 %s\n\n"+
			"👤 Плательщик: %s\n"+
			"🏢 Получатель: %s\n\n"+
			"🧾 Назначение:\n%s\n\n"+
			"💰 Сумма: %.0f %s\n"+
			"📅 Дата: %s",
		`АО "Модульбанк"`,
		payload.Operation.ContragentName,
		recipientName,
		payload.Operation.PaymentPurpose,
		payload.Operation.Amount,
		payload.Operation.Currency,
		payload.Operation.Executed,
	)

	tg.SendMessageInTelegramGroup(message)

	log.Printf("%+v\n", payload)

	w.WriteHeader(http.StatusOK)
}
