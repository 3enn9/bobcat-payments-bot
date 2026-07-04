package handlers

import (
	"PaymentsBot/internal/banks"
	"PaymentsBot/internal/domain/payment"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"io"
	"log"
	"math/big"
	"net/http"
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

func NewBanksHandler(service *banks.BankService) *BanksHandler {
	return &BanksHandler{banksSvc: *service}
}

type BanksHandler struct {
	banksSvc banks.BankService
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
		return
	}

	var modulBankPayment payment.ModulbankPayment
	err = json.Unmarshal(bodyBytes, &modulBankPayment)
	if err != nil {
		return
	}

	err = b.banksSvc.ModuleBank(modulBankPayment)
	if err != nil {
		log.Println(err)
		return
	}
	w.WriteHeader(http.StatusOK)
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
		return
	}

	var tBankPayment payment.TBankPayment
	if err = json.Unmarshal(bodyBytes, &tBankPayment); err != nil {
		return
	}

	err = b.banksSvc.TBank(tBankPayment)
	if err != nil {
		log.Println(err)
		return
	}

	w.WriteHeader(http.StatusOK)
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
		return
	}

	if len(body) == 0 {
		return
	}

	keyJSON := `{"kty":"RSA","e":"AQAB","n":"rwm77av7GIttq-JF1itEgLCGEZW_zz16RlUQVYlLbJtyRSu61fCec_rroP6PxjXU2uLzUOaGaLgAPeUZAJrGuVp9nryKgbZceHckdHDYgJd9TsdJ1MYUsXaOb9joN9vmsCscBx1lwSlFQyNQsHUsrjuDk-opf6RCuazRQ9gkoDCX70HV8WBMFoVm-YWQKJHZEaIQxg_DU4gMFyKRkDGKsYKA0POL-UgWA1qkg6nHY5BOMKaqxbc5ky87muWB5nNk4mfmsckyFv9j1gBiXLKekA_y4UwG2o1pbOLpJS3bP_c95rm4M9ZBmGXqfOQhbjz8z-s9C11i-jmOQ2ByohS-ST3E5sqBzIsxxrxyQDTw--bZNhzpbciyYW4GfkkqyeYoOPd_84jPTBDKQXssvj8ZOj2XboS77tvEO1n1WlwUzh8HPCJod5_fEgSXuozpJtOggXBv0C2ps7yXlDZf-7Jar0UYc_NJEHJF-xShlqd6Q3sVL02PhSCM-ibn9DN9BKmD"}`
	var jwk payment.JWK
	if err := json.Unmarshal([]byte(keyJSON), &jwk); err != nil {
		return
	}

	pubKey, err := jwkToPublicKey(jwk)
	if err != nil {
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
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		log.Printf("invalid claims %v", err)
		return
	}

	payloadBytes, err := json.Marshal(claims)
	if err != nil {
		log.Printf("cannot marshal claims %v", err)
		return
	}

	var tochkaPayment payment.TochkaPayment
	if err := json.Unmarshal(payloadBytes, &tochkaPayment); err != nil {
		log.Printf("cannot parse payment %v", err)
		return
	}

	err = b.banksSvc.TochkaBank(tochkaPayment)
	if err != nil {
		log.Println(err)
		return
	}

	w.WriteHeader(http.StatusOK)
}
