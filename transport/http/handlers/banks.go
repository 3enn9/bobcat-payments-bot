package handlers

import (
	"PaymentsBot/internal/db"
	"PaymentsBot/internal/domain/payment"
	"PaymentsBot/internal/webhookqueue"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"io"
	"log"
	"math/big"
	"net/http"
	"strings"
)

func jwkToPublicKey(jwk payment.JWK) (*rsa.PublicKey, error) {
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

func NewBanksHandler(queue *webhookqueue.Queue) *BanksHandler {
	return &BanksHandler{queue: queue}
}

type BanksHandler struct {
	queue *webhookqueue.Queue
}

func (b *BanksHandler) enqueuePayment(w http.ResponseWriter, source db.IncomingPaymentSource, externalID string, payload any) {
	externalID = strings.TrimSpace(externalID)
	if externalID == "" || len(externalID) > 128 {
		http.Error(w, "payment id is required", http.StatusBadRequest)
		return
	}
	if b.queue == nil {
		log.Printf("incoming payment queue unavailable source=%s external_id=%q", source, externalID)
		http.Error(w, "payment queue unavailable", http.StatusInternalServerError)
		return
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("encode incoming payment webhook source=%s external_id=%q: %v", source, externalID, err)
		http.Error(w, "failed to accept incoming payment", http.StatusInternalServerError)
		return
	}
	if err := b.queue.Enqueue(string(source), externalID, payloadBytes); err != nil {
		log.Printf("enqueue incoming payment failed source=%s external_id=%q: %v", source, externalID, err)
		http.Error(w, "failed to accept incoming payment", http.StatusInternalServerError)
		return
	}
	log.Printf("incoming payment webhook accepted source=%s external_id=%q", source, externalID)
	w.WriteHeader(http.StatusOK)
}

// ModuleBankHandler обработчик веб хука Модуль банка
func (b *BanksHandler) ModuleBankHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read webhook body", http.StatusInternalServerError)
		return
	}

	var modulBankPayment payment.ModulbankPayment
	err = json.Unmarshal(bodyBytes, &modulBankPayment)
	if err != nil {
		http.Error(w, "invalid webhook payload", http.StatusBadRequest)
		return
	}

	if modulBankPayment.Operation.Category != "Debet" {
		w.WriteHeader(http.StatusOK)
		return
	}
	b.enqueuePayment(w, db.SourceModulbank, modulBankPayment.Operation.ID, modulBankPayment)
}

// TBankHandler Обработчик веб хука Тбанка
func (b *BanksHandler) TBankHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read webhook body", http.StatusInternalServerError)
		return
	}

	var tBankPayment payment.TBankPayment
	if err = json.Unmarshal(bodyBytes, &tBankPayment); err != nil {
		http.Error(w, "invalid webhook payload", http.StatusBadRequest)
		return
	}

	if tBankPayment.TypeOfOperation == "Debit" {
		w.WriteHeader(http.StatusOK)
		return
	}
	b.enqueuePayment(w, db.SourceTbank, tBankPayment.OperationID, tBankPayment)
}

func (b *BanksHandler) TochkaBankHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println("read body error:", err)
		http.Error(w, "failed to read webhook body", http.StatusInternalServerError)
		return
	}

	if len(body) == 0 {
		http.Error(w, "empty webhook body", http.StatusBadRequest)
		return
	}

	keyJSON := `{"kty":"RSA","e":"AQAB","n":"rwm77av7GIttq-JF1itEgLCGEZW_zz16RlUQVYlLbJtyRSu61fCec_rroP6PxjXU2uLzUOaGaLgAPeUZAJrGuVp9nryKgbZceHckdHDYgJd9TsdJ1MYUsXaOb9joN9vmsCscBx1lwSlFQyNQsHUsrjuDk-opf6RCuazRQ9gkoDCX70HV8WBMFoVm-YWQKJHZEaIQxg_DU4gMFyKRkDGKsYKA0POL-UgWA1qkg6nHY5BOMKaqxbc5ky87muWB5nNk4mfmsckyFv9j1gBiXLKekA_y4UwG2o1pbOLpJS3bP_c95rm4M9ZBmGXqfOQhbjz8z-s9C11i-jmOQ2ByohS-ST3E5sqBzIsxxrxyQDTw--bZNhzpbciyYW4GfkkqyeYoOPd_84jPTBDKQXssvj8ZOj2XboS77tvEO1n1WlwUzh8HPCJod5_fEgSXuozpJtOggXBv0C2ps7yXlDZf-7Jar0UYc_NJEHJF-xShlqd6Q3sVL02PhSCM-ibn9DN9BKmD"}`
	var jwk payment.JWK
	if err := json.Unmarshal([]byte(keyJSON), &jwk); err != nil {
		http.Error(w, "webhook verification unavailable", http.StatusInternalServerError)
		return
	}

	pubKey, err := jwkToPublicKey(jwk)
	if err != nil {
		http.Error(w, "webhook verification unavailable", http.StatusInternalServerError)
		return
	}

	token, err := jwt.Parse(string(body), func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return pubKey, nil
	})
	if err != nil || !token.Valid {
		log.Printf("invalid signature %v", err)
		http.Error(w, "invalid webhook signature", http.StatusUnauthorized)
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		log.Printf("invalid claims %v", err)
		http.Error(w, "invalid webhook claims", http.StatusBadRequest)
		return
	}

	payloadBytes, err := json.Marshal(claims)
	if err != nil {
		log.Printf("cannot marshal claims %v", err)
		http.Error(w, "invalid webhook claims", http.StatusBadRequest)
		return
	}

	var tochkaPayment payment.TochkaPayment
	if err := json.Unmarshal(payloadBytes, &tochkaPayment); err != nil {
		log.Printf("cannot parse payment %v", err)
		http.Error(w, "invalid webhook payload", http.StatusBadRequest)
		return
	}

	b.enqueuePayment(w, db.SourceTochka, tochkaPayment.PaymentId, tochkaPayment)
}
