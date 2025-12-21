package banks

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
)

type TBankPayment struct {
	OperationID     string `json:"operationId"`
	TypeOfOperation string `json:"typeOfOperation"`
	AccountNumber   string `json:"accountNumber"`
	DocumentNumber  string `json:"documentNumber"`

	OperationAmount              string `json:"operationAmount"`
	OperationCurrencyDigitalCode string `json:"operationCurrencyDigitalCode"`

	AccountAmount              string `json:"accountAmount"`
	AccountCurrencyDigitalCode string `json:"accountCurrencyDigitalCode"`

	RubleAmount string `json:"rubleAmount"`

	CounterParty CounterParty `json:"counterParty"`

	Description string `json:"description"`

	AuthorizationDate string `json:"authorizationDate"`
	TrxnPostDate      string `json:"trxnPostDate"`

	PayVo    string `json:"payVo"`
	Priority string `json:"priority"`

	CardNumber string `json:"cardNumber"`
	Ucid       string `json:"ucid"`
	Mcc        string `json:"mcc"`

	Merch Merch `json:"merch"`

	Status          string `json:"status"`
	OperationStatus string `json:"operationStatus"`
	Bic             string `json:"bic"`
	Rrn             string `json:"rrn"`
	Category        string `json:"category"`

	PayPurpose string `json:"payPurpose"`

	Receiver Party `json:"receiver"`
	Payer    Party `json:"payer"`

	ChargeDate string `json:"chargeDate"`
	DrawDate   string `json:"drawDate"`

	Kbk          string `json:"kbk"`
	Oktmo        string `json:"oktmo"`
	TaxEvidence  string `json:"taxEvidence"`
	TaxPeriod    string `json:"taxPeriod"`
	TaxDocNumber string `json:"taxDocNumber"`
	TaxDocDate   string `json:"taxDocDate"`

	NalType string `json:"nalType"`

	DocDate string `json:"docDate"`
	VO      string `json:"VO"`
}

type CounterParty struct {
	Account     string `json:"account"`
	BankBic     string `json:"bankBic"`
	BankName    string `json:"bankName"`
	CorrAccount string `json:"corrAccount"`
	Inn         string `json:"inn"`
	Name        string `json:"name"`
}

type Party struct {
	Account     string `json:"account"`
	Name        string `json:"name"`
	Inn         string `json:"inn"`
	Bic         string `json:"bic"`
	CorrAccount string `json:"corrAccount"`
	BankName    string `json:"bankName"`
}

type Merch struct {
	Address string `json:"address"`
	City    string `json:"city"`
	Country string `json:"country"`
	Index   string `json:"index"`
	Name    string `json:"name"`
}

func TBankHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	defer r.Body.Close()

	bodyBytes, err := io.ReadAll(r.Body)

	if err != nil {
		log.Printf("tbank: read body error %v", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	var payment TBankPayment

	if err = json.Unmarshal(bodyBytes, &payment); err != nil {
		log.Println("Error unmarshaling tbank", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	log.Printf("TBank %+v", payment)

	w.WriteHeader(http.StatusOK)
}
