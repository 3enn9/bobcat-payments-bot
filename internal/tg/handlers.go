package tg

import (
	"PaymentsBot/internal/db"
	"fmt"
	"log"
	"strconv"
	"strings"
)

func (s *TelegramService) handleAdd(chatID int64, text, chatName string) {

	operationArray := strings.Split(text, "\n")

	var operations []string
	var errorsList []string
	var totalAmount float64

	for i, operation := range operationArray {
		parts := strings.Fields(operation)

		if len(parts) < 2 {
			errorsList = append(errorsList, operation+" (неверный формат)")
			continue
		}

		if i == 0 {
			parts = parts[1:]
		}

		amountStr := parts[len(parts)-1]
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			errorsList = append(errorsList, operation+" (неверная сумма)")
			continue
		}
		description := strings.Join(parts[:len(parts)-1], " ")
		operationType := "withdraw"
		if amount >= 0 {
			operationType = "deposit"
		}

		payment := db.Payment{Description: description, Operation: operationType, TelegramGroupID: chatID, Title: chatName, Amount: amount}

		err = s.db.AddPayment(payment)

		if err != nil {
			log.Printf("Не удалось добавить операцию %v", err)
			continue
		}
		err = s.db.UpdateBalance(chatID, chatName, amount)
		if err != nil {
			errorsList = append(errorsList, operation+" (ошибка БД)")
			continue
		}

		operations = append(operations, fmt.Sprintf("• %s: %.2f", description, amount))
		totalAmount += amount

		if amount > 0 {
			cashMessage := fmt.Sprintf("%s\n💬 %s\n💰 Сумма: %.2f\n", chatName, description, amount)
			s.SendMessageInTelegramGroup(s.Chats["Cash"], cashMessage)
		}
	}
	balance, err := s.db.GetBalance(chatID)
	if err != nil {
		log.Printf("Ошибка получения баланса: %v", err)
	}

	msg := fmt.Sprintf(
		"📊 Операции:\n%s\n\n💰 Итого: %.2f\n🏦 Касса: %.2f",
		strings.Join(operations, "\n"),
		totalAmount,
		balance,
	)

	if len(errorsList) > 0 {
		msg += "\n\n⚠ Пропущены:\n" + strings.Join(errorsList, "\n")
	}

	s.SendMessageInTelegramGroup(chatID, msg)
}

func (s *TelegramService) handleAll(chatID int64) {
	msg, err := s.db.AllBalance()
	if err != nil {
		log.Printf("error connection: %v", err)
		return
	}

	if msg == "" {
		msg = "Нет данных"
	}
	s.SendMessageInTelegramGroup(chatID, msg)
}

func (s *TelegramService) handleDeposit(chatID int64, text, chatName string) {

}

func (s *TelegramService) handleSalary(chatID int64) {

}
