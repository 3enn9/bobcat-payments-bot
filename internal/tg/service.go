package tg

import (
	"PaymentsBot/internal/db"
	"PaymentsBot/internal/domain/messenger"
	"PaymentsBot/internal/domain/payment"
	"PaymentsBot/internal/usecase"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"strconv"
	"strings"
)

type TelegramService struct {
	bot   *tgbotapi.BotAPI
	Chats map[string]int64
	db    usecase.Repository
}

func NewTelegramService(token string, db *db.Database) (*TelegramService, error) {
	bot, err := tgbotapi.NewBotAPI(token)

	if err != nil {
		return nil, err
	}

	chats := map[string]int64{
		"Payments": -1003380906513,
		"Fuels":    -1003368403742,
		"Cash":     -1003797529492,
	}

	return &TelegramService{bot, chats, db}, nil
}

func (s *TelegramService) SendMessageInGroupID(chatID int64, message string) error {

	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = "HTML"

	sentMsg, err := s.bot.Send(msg)
	if err != nil {
		return messenger.ErrSendMessage
	}

	if strings.Contains(message, "Касса:") {
		pin := tgbotapi.PinChatMessageConfig{ChatID: chatID, MessageID: sentMsg.MessageID, DisableNotification: true}
		resp, err := s.bot.Request(pin)
		if err != nil {
			fmt.Printf("Error pin message")
			return messenger.ErrPinMessage
		}
		fmt.Println(resp.Description)

	}
	return nil
}

func (s *TelegramService) SendMessageInGroupName(nameGroup string, message string) error {
	chatID, ok := s.Chats[nameGroup]
	if !ok{
		return fmt.Errorf("groupNmae not in map chats")
	}
	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = "HTML"

	_, err := s.bot.Send(msg)
	if err != nil {
		return messenger.ErrSendMessage
	}

	return nil
}


func (s *TelegramService) Updates(u tgbotapi.Update) error {
	if u.Message == nil {
		return messenger.ErrMessageEmpty
	}
	text := u.Message.Text
	chatID := u.Message.Chat.ID
	chatName := u.Message.Chat.Title

	switch {
	case strings.HasPrefix(text, "/add "):
		s.AddPayment(chatID, text, chatName)
	case strings.HasPrefix(text, "/all "):
		s.AllBalance(chatID)
	case strings.HasPrefix(text, "/dep "):
		s.Deposit(chatID, text, chatName)
	case strings.HasPrefix(text, "/salary "):
		s.Salary(chatID)
	}

	return nil
}

func (s *TelegramService) GetGroupID(groupName string) (int64, error) {
	id, ok := s.Chats[groupName]
	if !ok {
		return 0, messenger.ErrGroupExists
	}
	return id, nil
}

func (s *TelegramService) Balance(chatID int64) (int64, error) {
	return 0, nil
}

func (s *TelegramService) AddPayment(chatID int64, text, chatName string) error {

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

		pymt := payment.Payment{
			Description: description,
			Operation:   operationType,
			GroupID:     chatID,
			Title:       chatName,
			Amount:      amount,
		}

		err = s.db.AddPayment(pymt)

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
			err = s.SendMessageInGroupName("Cash", cashMessage)
			if err != nil {
				return err
			}
		}
	}
	balance, err := s.db.GetBalance(chatID)
	if err != nil {
		return fmt.Errorf("error GetBalance: %v", err)
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

	err = s.SendMessageInGroupID(chatID, msg)
	if err != nil {
		return err
	}
	return err
}

func (s *TelegramService) AllBalance(chatID int64) error {
	msg, err := s.db.AllBalance()
	if err != nil {
		return fmt.Errorf("error connection: %v", err)
	}

	if msg == "" {
		msg = "Нет данных"
	}
	err = s.SendMessageInGroupID(chatID, msg)
	if err != nil {
		return err
	}
	return nil
}

func (s *TelegramService) Deposit(chatID int64, text, chatName string) error {
	return nil
}

func (s *TelegramService) Salary(chatID int64) error {
	return nil
}
