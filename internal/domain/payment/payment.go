package payment

type Payment struct {
	GroupID     int64
	Title       string
	Operation   string
	Description string
	Amount      float64
}

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

type ModulbankPayment struct {
	CompanyInn string    `json:"companyInn"`
	CompanyKpp string    `json:"companyKpp"`
	Operation  Operation `json:"operation"`
	SHA1Hash   string    `json:"SHA1Hash"`
}

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

type TochkaPayment struct {
	SidePayer struct {
		BankCode                 string `json:"bankCode"`
		BankName                 string `json:"bankName"`
		BankCorrespondentAccount string `json:"bankCorrespondentAccount"`
		Account                  string `json:"account"`
		Name                     string `json:"name"`
		Amount                   string `json:"amount"`
		Currency                 string `json:"currency"`
		Inn                      string `json:"inn"`
		Kpp                      string `json:"kpp"`
	} `json:"SidePayer"`
	SideRecipient struct {
		BankCode                 string `json:"bankCode"`
		BankName                 string `json:"bankName"`
		BankCorrespondentAccount string `json:"bankCorrespondentAccount"`
		Account                  string `json:"account"`
		Name                     string `json:"name"`
		Amount                   string `json:"amount"`
		Currency                 string `json:"currency"`
		Inn                      string `json:"inn"`
		Kpp                      string `json:"kpp"`
	} `json:"SideRecipient"`
	Purpose        string `json:"purpose"`
	DocumentNumber string `json:"documentNumber"`
	PaymentId      string `json:"paymentId"`
	Date           string `json:"date"`
	WebhookType    string `json:"webhookType"`
	CustomerCode   string `json:"customerCode"`
}

type JWK struct {
	Kty string `json:"kty"`
	N   string `json:"n"`
	E   string `json:"e"`
}
