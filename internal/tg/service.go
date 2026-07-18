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
	bot      *tgbotapi.BotAPI
	Chats    map[string]int64
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
			return nil
		}
		fmt.Println(resp.Description)

	}
	return nil
}

func (s *TelegramService) SendMessageInGroupName(nameGroup string, message string) error {
	chatID, ok := s.Chats[nameGroup]
	if !ok {
		return fmt.Errorf("groupNmae not in map chats")
	}
	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = "HTML"

	_, err := s.bot.Send(msg)
	if err != nil {
		fmt.Printf("Error send message")
		return nil
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
		result, err := s.payments.AddPayment(cashCh, chatID, text, chatName)

		if err != nil {
			log.Println("error AddPayments")
			return nil
		}
		err = s.SendMessageInGroupID(chatID, result.GroupMessage)
		if err != nil {
			log.Printf("error send message in groupID %v", err)
		}
		err = s.SendMessageInGroupName("Cash", result.CashMessage)
		if err != nil {
			log.Printf("error send message in groupName %v", err)
		}
	case strings.HasPrefix(text, "/all"):
		msg, err := s.payments.AllBalance(chatID)
		if err != nil {
			log.Printf("error AllBalance %v", err)
			return nil
		}
		err = s.SendMessageInGroupID(chatID, msg)
		if err != nil {
			log.Printf("error send message in groupID %v", err)
		}
	case strings.HasPrefix(text, "/dep "):
		err := s.payments.Deposit(chatID, text, chatName)
		if err != nil {
			log.Printf("error deposit %v", err)
		}
	case strings.HasPrefix(text, "/salary "):
		err := s.payments.Salary(chatID)
		if err != nil {
			log.Printf("error salary %v", err)
		}
	}

	return nil
}
