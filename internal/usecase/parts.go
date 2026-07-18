package usecase

import (
	"PaymentsBot/internal/domain/payment"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Repository interface {
	UpdateBalance(chatID int64, title string, amount float64) error
	GetBalance(chatID int64) (float64, error)
	AllBalance() (string, error)
	AddPayment(payment payment.Payment) error
}

type UseCase interface {
	UpdateBalance(chatID int64, title string, amount float64) error
	GetBalance(chatID int64) (float64, error)
	AllBalance() (string, error)
	AddPayment(input CreateInput) error
}

type Messenger interface {
	SendMessageInGroupID(chatID int64, message string) error
	SendMessageInGroupName(groupName string, message string) error
	Updates(u tgbotapi.Update) error // Поменять на интерфейс с добавлением других мессенджеров
	GetGroupID(groupName string) (int64, error)
}

type SendMessanger interface {
	SendMessageInGroupName(groupName string, message string) error
	SendMessageInGroupID(chatID int64, message string) error
}

type CreateInput struct {
	TelegramGroupID int64
	Title           string
	Operation       string
	Description     string
	Amount          float64
}
