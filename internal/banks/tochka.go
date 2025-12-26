package banks

import (
	"PaymentsBot/internal/tg"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"io"
	"log"
	"math/big"
	"net/http"
	"time"
)

type incomingPayment struct {
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

func DateFormatTochka(date string) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	} else {
		return t.Format("02.01.2006")
	}

}

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

func TochkaBankHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "error read body", http.StatusBadRequest)
		log.Println("read body error:", err)
		return
	}

	if len(body) == 0 {
		http.Error(w, "empty body", http.StatusBadRequest)
		log.Println("empty body")
		return
	}

	// Публичный ключ Точки
	keyJSON := `{"kty":"RSA","e":"AQAB","n":"rwm77av7GIttq-JF1itEgLCGEZW_zz16RlUQVYlLbJtyRSu61fCec_rroP6PxjXU2uLzUOaGaLgAPeUZAJrGuVp9nryKgbZceHckdHDYgJd9TsdJ1MYUsXaOb9joN9vmsCscBx1lwSlFQyNQsHUsrjuDk-opf6RCuazRQ9gkoDCX70HV8WBMFoVm-YWQKJHZEaIQxg_DU4gMFyKRkDGKsYKA0POL-UgWA1qkg6nHY5BOMKaqxbc5ky87muWB5nNk4mfmsckyFv9j1gBiXLKekA_y4UwG2o1pbOLpJS3bP_c95rm4M9ZBmGXqfOQhbjz8z-s9C11i-jmOQ2ByohS-ST3E5sqBzIsxxrxyQDTw--bZNhzpbciyYW4GfkkqyeYoOPd_84jPTBDKQXssvj8ZOj2XboS77tvEO1n1WlwUzh8HPCJod5_fEgSXuozpJtOggXBv0C2ps7yXlDZf-7Jar0UYc_NJEHJF-xShlqd6Q3sVL02PhSCM-ibn9DN9BKmD"}`
	var jwk JWK
	if err := json.Unmarshal([]byte(keyJSON), &jwk); err != nil {
		http.Error(w, "invalid JWK", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	pubKey, err := jwkToPublicKey(jwk)
	if err != nil {
		http.Error(w, "cannot parse public key", http.StatusInternalServerError)
		log.Println(err)
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
		log.Printf("invalid signature %v", err)
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		http.Error(w, "invalid claims", http.StatusBadRequest)
		log.Printf("invalid claims %v", err)
		return
	}

	payloadBytes, err := json.Marshal(claims)
	if err != nil {
		http.Error(w, "cannot marshal claims", http.StatusInternalServerError)
		log.Printf("cannot marshal claims %v", err)
		return
	}

	var payment incomingPayment
	if err := json.Unmarshal(payloadBytes, &payment); err != nil {
		http.Error(w, "cannot parse payment", http.StatusBadRequest)
		log.Printf("cannot parse payment %v", err)
		return
	}

	fmt.Printf("Получен платеж: %+v\n", payment)

	date := DateFormatTochka(payment.Date)

	message := fmt.Sprintf(
		"🏦 %s\n\n"+
			"👤 Плательщик: %s\n"+
			"🏢 Получатель: %s\n\n"+
			"🧾 Назначение:\n%s\n\n"+
			"💰 Оплата:\n"+
			"```%s %s %s```",
		payment.SideRecipient.BankName,
		payment.SidePayer.Name,
		payment.SideRecipient.Name,
		payment.Purpose,
		date,
		payment.SidePayer.Amount,
		"точка",
	)

	tg.SendMessageInTelegramGroup(message)

	w.WriteHeader(http.StatusOK)
}
