package rncard

import (
	"PaymentsBot/internal/tg"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type apiResponse struct {
	OperationList []operation `json:"OperationList"`
}

type operation struct {
	Ref    string  `json:"Ref"`
	Code   string  `'json:"Code"`
	Sum    float64 `json:"Sum"`
	Value  float64 `json:"Value"`
	Holder string  `json:"Holder"`
	GName  string  `json:"GName"`
	Date   string  `json:"Date"`
}

var TgBot *tg.TelegramService

func SetTelegram(t *tg.TelegramService) {
	TgBot = t
}

func FetchAndSendTransactions() error {

	var message string

	date := time.Now().AddDate(0, 0, -1)

	begin := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	end := time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 0, date.Location())

	beginStr := begin.Format("2006-01-02T15:04:05")
	endStr := end.Format("2006-01-02T15:04:05")

	log.Printf("Begin %s - end %s", beginStr, endStr)

	baseURL := "https://lkapi.rn-card.ru/api/emv/v2/GetOperByContract"

	params := url.Values{}
	params.Set("u", "Kal9n")
	params.Set("p", "Kal9n474788")
	params.Set("contract", "ISS163200")
	params.Set("begin", beginStr)
	params.Set("end", endStr)
	params.Set("type", "JSON")

	reqURL := baseURL + "?" + params.Encode()

	resp, err := http.Get(reqURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Bad request: %s", resp.Status)
	}

	var apiResp apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("error decoder %v", err)
	}

	operations := apiResp.OperationList
	indexesToRemove := map[int]bool{}

	for i, op := range operations {
		if op.Ref != "" && op.Ref != op.Code {
			for j, op2 := range operations {
				if op2.Code == op.Ref {
					operations[j].Sum -= op.Sum
					operations[j].Value -= op.Value
					indexesToRemove[i] = true
					break
				}
			}
		}
	}

	filtered := []operation{}
	for i, op := range operations {
		if !indexesToRemove[i] {
			filtered = append(filtered, op)
		}
	}

	for _, op := range filtered {
		parsedDate, err := time.Parse("2006-01-02T15:04:05", op.Date)
		if err != nil {
			continue
		}

		formattedDate := parsedDate.Format("02.01.2006 15:04:05")
		sumStr := strings.Replace(
			strconv.FormatFloat(op.Sum, 'f', 2, 64),
			".",
			",",
			1,
		)

		holder := strings.ReplaceAll(op.Holder, " ", "-")

		operationInfo := fmt.Sprintf(
			"%s %s %s %s",
			formattedDate,
			sumStr,
			holder,
			op.GName,
		)
		message += operationInfo

	}
	TgBot.SendMessageInTelegramGroup("Fuels", message)

	log.Printf("fuels message: %s", message)

	return nil
}
