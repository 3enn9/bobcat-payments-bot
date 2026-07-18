package max

import (
	"PaymentsBot/internal/payments"
	"context"
	"fmt"
	maxbot "github.com/max-messenger/max-bot-api-client-go"
	"github.com/max-messenger/max-bot-api-client-go/schemes"
	"log"
)

type MaxService struct {
	Bot      *maxbot.Api
	Chats    map[string]int64
	payments *payments.PaymentsService
}

func NewMaxService(token string, payments *payments.PaymentsService) (*MaxService, error) {
	api, err := maxbot.New(token)
	if err != nil {
		return nil, err
	}
	chats := map[string]int64{
		"Payments": -77028763384544,
		"Fuels":    -77028768299744,
		"Cash":     -77028778785504}
	return &MaxService{Bot: api, Chats: chats, payments: payments}, nil
}

func (m *MaxService) SendMessageInGroupID(chatID int64, message string) error {
	msg := maxbot.NewMessage().SetChat(chatID).SetText(message)
	err := m.Bot.Messages.Send(context.Background(), msg)
	if err != nil {
		fmt.Printf("error send message %v", err)
	}
	return nil
}
func (m *MaxService) SendMessageInGroupName(nameGroup string, message string) error {
	chatID, ok := m.Chats[nameGroup]
	if !ok {
		log.Println("group name does not exists")
		return nil
	}

	msg := maxbot.NewMessage().SetChat(chatID).SetText(message)
	err := m.Bot.Messages.Send(context.Background(), msg)
	if err != nil {
		log.Printf("error send message %v", err)
	}
	return nil
}

func (m *MaxService) Updates(update schemes.UpdateInterface) error {
	switch upd := update.(type) {

	case *schemes.MessageCreatedUpdate:
		m.handleMessage(upd)

	default:
		log.Printf("unknown update %T", upd)
	}

	return nil
}

func (m *MaxService) handleMessage(upd *schemes.MessageCreatedUpdate) {
	//command := strings.Fields(upd.Message.Body.Text)[0]

	//fmt.Printf("inbox command %s", command)
	fmt.Printf("chatID from message %v", upd.GetChatID())
}
