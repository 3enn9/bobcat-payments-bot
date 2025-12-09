package main

import (
	"crypto/rsa"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/golang-jwt/jwt/v5"
)

// IncomingPayment Структура платежа (incomingPayment)
type IncomingPayment struct {
	SidePayer struct {
		BankCode                 string  `json:"bankCode"`
		BankName                 string  `json:"bankName"`
		BankCorrespondentAccount string  `json:"bankCorrespondentAccount"`
		Account                  string  `json:"account"`
		Name                     string  `json:"name"`
		Amount                   string  `json:"amount"`
		Currency                 float64 `json:"currency"`
		Inn                      string  `json:"inn"`
		Kpp                      string  `json:"kpp"`
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

type ModulbankWebhook struct {
	CompanyInn string    `json:"companyInn"`
	CompanyKpp string    `json:"companyKpp"`
	Operation  Operation `json:"operation"`
	SHA1Hash   string    `json:"SHA1Hash"`
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

// JWK Структура JWK (публичный ключ Точки)
type JWK struct {
	Kty string `json:"kty"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func SendMessageInTelegramGroup(message string) {
	bot, err := tgbotapi.NewBotAPI("8440241939:AAEvMsPT9FeOFWlvexZfvmxg9GcOxXoR7yE")

	if err != nil {
		log.Panic(err)
	}

	chatID := int64(-1003380906513)
	msg := tgbotapi.NewMessage(chatID, message)

	_, err = bot.Send(msg)
	if err != nil {
		log.Panic(err)
	}
	log.Println("Сообщение отправлено")
}

// Преобразуем JWK в *rsa.PublicKey
func jwkToPublicKey(jwk JWK) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, err
	}

	eInt := 0
	for _, b := range eBytes {
		eInt = eInt<<8 + int(b)
	}

	pubKey := &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: eInt,
	}

	return pubKey, nil
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body := make([]byte, r.ContentLength)
	r.Body.Read(body)

	// Публичный ключ Точки
	keyJSON := `{"kty":"RSA","e":"AQAB","n":"rwm77av7GIttq-JF1itEgLCGEZW_zz16RlUQVYlLbJtyRSu61fCec_rroP6PxjXU2uLzUOaGaLgAPeUZAJrGuVp9nryKgbZceHckdHDYgJd9TsdJ1MYUsXaOb9joN9vmsCscBx1lwSlFQyNQsHUsrjuDk-opf6RCuazRQ9gkoDCX70HV8WBMFoVm-YWQKJHZEaIQxg_DU4gMFyKRkDGKsYKA0POL-UgWA1qkg6nHY5BOMKaqxbc5ky87muWB5nNk4mfmsckyFv9j1gBiXLKekA_y4UwG2o1pbOLpJS3bP_c95rm4M9ZBmGXqfOQhbjz8z-s9C11i-jmOQ2ByohS-ST3E5sqBzIsxxrxyQDTw--bZNhzpbciyYW4GfkkqyeYoOPd_84jPTBDKQXssvj8ZOj2XboS77tvEO1n1WlwUzh8HPCJod5_fEgSXuozpJtOggXBv0C2ps7yXlDZf-7Jar0UYc_NJEHJF-xShlqd6Q3sVL02PhSCM-ibn9DN9BKmD"}`
	var jwk JWK
	if err := json.Unmarshal([]byte(keyJSON), &jwk); err != nil {
		http.Error(w, "invalid JWK", http.StatusInternalServerError)
		return
	}

	pubKey, err := jwkToPublicKey(jwk)
	if err != nil {
		http.Error(w, "cannot parse public key", http.StatusInternalServerError)
		return
	}

	// Проверяем JWT
	token, err := jwt.Parse(string(body), func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return pubKey, nil
	})
	if err != nil || !token.Valid {
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	// Парсим payload из JWT
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		http.Error(w, "invalid claims", http.StatusBadRequest)
		return
	}

	payloadBytes, err := json.Marshal(claims)
	if err != nil {
		http.Error(w, "cannot marshal claims", http.StatusInternalServerError)
		return
	}

	var payment IncomingPayment
	if err := json.Unmarshal(payloadBytes, &payment); err != nil {
		http.Error(w, "cannot parse payment", http.StatusBadRequest)
		return
	}

	fmt.Printf("Получен платеж: %+v\n", payment)

	message := fmt.Sprintf(
		"🏦 %s\n\n"+
			"👤 Плательщик: %s\n"+
			"🏢 Получатель: %s\n\n"+
			"🧾 Назначение:\n%s\n\n"+
			"💰 Сумма: %.0f %s\n"+
			"📅 Дата: %s",
		payment.SideRecipient.BankName,
		payment.SidePayer.Name,
		payment.SideRecipient.Name,
		payment.Purpose,
		payment.SidePayer.Currency,
		payment.SidePayer.Amount,
		payment.Date,
	)

	SendMessageInTelegramGroup(message)

	w.WriteHeader(http.StatusOK)
}

func checkSHA1(body []byte, secret string) (bool, error) {
	var payload struct {
		SHA1Hash string `json:"SHA1Hash"`
	}

	// Достаём SHA1Hash
	if err := json.Unmarshal(body, &payload); err != nil {
		return false, err
	}

	if payload.SHA1Hash == "" {
		return false, errors.New("SHA1Hash not found")
	}

	// Удаляем поле "SHA1Hash" из JSON строкой
	raw := string(body)

	raw = strings.ReplaceAll(
		raw,
		`,"SHA1Hash":"`+payload.SHA1Hash+`"`,
		"",
	)

	raw = strings.ReplaceAll(
		raw,
		`"SHA1Hash":"`+payload.SHA1Hash+`",`,
		"",
	)

	// Считаем SHA1
	h := sha1.New()
	h.Write([]byte(raw + secret))
	localHash := hex.EncodeToString(h.Sum(nil))

	return localHash == payload.SHA1Hash, nil
}

func moduleBankHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}

	//secret := "MGM5OTBjNmEtOTRiNy00YzdhLWEwMmItYmNmMDAwYTBiNWU5MDE3MWU1NmMtN2Y3Ni00OTllLThkM2UtOTgyNzhhMTg3ZDRl"

	//ok, err := checkSHA1(bodyBytes, secret)
	//if err != nil || !ok {
	//	http.Error(w, "invalid signature", http.StatusForbidden)
	//	log.Println("Не прошел проверку SHA1")
	//	return
	//}

	var payload ModulbankWebhook
	err = json.Unmarshal(bodyBytes, &payload)
	if err != nil {
		http.Error(w, "error marshaling", http.StatusBadRequest)
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

	SendMessageInTelegramGroup(message)

	log.Printf("%+v\n", payload)

	w.WriteHeader(http.StatusOK)
}

func TBankHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func main() {
	http.HandleFunc("/webhook", webhookHandler)
	http.HandleFunc("/modulbank", moduleBankHandler)
	http.HandleFunc("/tbank", TBankHandler)
	fmt.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
