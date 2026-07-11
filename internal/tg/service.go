package tg

import (
	"PaymentsBot/internal/domain/messenger"
	"PaymentsBot/internal/payments"
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TelegramService struct {
	bot   *tgbotapi.BotAPI
	Chats map[string]int64
	payments *payments.PaymentsService
}

func NewTelegramService(token string, payments *payments.PaymentsService) (*TelegramService, error) {
	bot, err := tgbotapi.NewBotAPI(token)

	if err != nil {
		return nil, err
	}

	chats := map[string]int64{
		"Payments": -1003380906513,
		"Fuels":    -1003368403742,
		"Cash":     -1003797529492,
	}

	return &TelegramService{bot, chats, payments}, nil
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
		cashCh := make(chan string, 1)
		err := s.payments.AddPayment(cashCh, chatID, text, chatName)
		

		if err != nil{
			log.Println("error AddPayments")
			return nil
		}
		s.SendMessageInGroupID(chatID, )
	case strings.HasPrefix(text, "/all "):
		msg, err := s.payments.AllBalance(chatID)
		if err != nil{
			log.Println("error AllBalance")
			return nil
		}
		s.SendMessageInGroupID(chatID, msg)
	case strings.HasPrefix(text, "/dep "):
		s.payments.Deposit(chatID, text, chatName)
	case strings.HasPrefix(text, "/salary "):
		s.payments.Salary(chatID)
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