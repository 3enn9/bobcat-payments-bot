package payments

import (
	"PaymentsBot/internal/domain/payment"
	"PaymentsBot/internal/usecase"
	"fmt"
	"log"
	"strconv"
	"strings"
)

type PaymentsService struct {
	db usecase.Repository
}

type AddPaymentResult struct {
	GroupMessage string
	CashMessage  string
}

func NewPaymentsService(db usecase.Repository) *PaymentsService {
	return &PaymentsService{db: db}
}

func (p *PaymentsService) Balance(chatID int64) (int64, error) {
	return 0, nil
}

func (p *PaymentsService) AddPayment(cashCh chan string, chatID int64, text, chatName string) (*AddPaymentResult, error) {

	operationArray := strings.Split(text, "\n")
	var cashMessages []string
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

		pymt := payment.Payment{
			Description: description,
			Operation:   operationType,
			GroupID:     chatID,
			Title:       chatName,
			Amount:      amount,
		}

		err = p.db.AddPayment(pymt)

		if err != nil {
			log.Printf("Не удалось добавить операцию %v", err)
			continue
		}
		err = p.db.UpdateBalance(chatID, chatName, amount)
		if err != nil {
			errorsList = append(errorsList, operation+" (ошибка БД)")
			continue
		}

		operations = append(operations, fmt.Sprintf("• %s: %.2f", description, amount))
		totalAmount += amount

		if amount > 0 {
			cashMessages = append(cashMessages,
				fmt.Sprintf("%s\n💬 %s\n💰 Сумма: %.2f",
					chatName, description, amount))
		}
	}
	balance, err := p.db.GetBalance(chatID)
	if err != nil {
		return nil, fmt.Errorf("error GetBalance: %v", err)
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

	return &AddPaymentResult{
		GroupMessage: msg,
		CashMessage:  strings.Join(cashMessages, "\n\n"),
	}, nil
}

func (p *PaymentsService) AllBalance(chatID int64) (string, error) {
	msg, err := p.db.AllBalance()
	if err != nil {
		return "", fmt.Errorf("error connection: %v", err)
	}

	if msg == "" {
		msg = "Нет данных"
	}
	return msg, nil
}

func (p *PaymentsService) Deposit(chatID int64, text, chatName string) error {
	return nil
}

func (p *PaymentsService) Salary(chatID int64) error {
	return nil
}
