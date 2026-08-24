package max

import (
	"PaymentsBot/internal/db"
	"PaymentsBot/internal/payments"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	maxbot "github.com/max-messenger/max-bot-api-client-go"
	"github.com/max-messenger/max-bot-api-client-go/schemes"
)

type MaxService struct {
	Bot      *maxbot.Api
	Chats    map[string]int64
	payments *payments.PaymentsService
	db       *db.Database
}

func NewMaxService(token string, payments *payments.PaymentsService, database *db.Database) (*MaxService, error) {
	api, err := maxbot.New(token)
	if err != nil {
		return nil, err
	}
	chats := map[string]int64{
		"Payments": -77028763384544,
		"Fuels":    -77028768299744,
		"Cash":     -77028778785504,
		"Rogatka":  -71392114984255, // TODO: заменить на chat_id «Рогатка заявки»
	}
	return &MaxService{Bot: api, Chats: chats, payments: payments, db: database}, nil
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
	text := strings.TrimSpace(upd.GetText())
	if text == "" {
		return
	}

	fields := strings.Fields(text)
	cmd := strings.Split(fields[0], "@")[0]

	switch cmd {
	case "/group":
		m.handleGroupCommand(upd)
		return
	}

	if strings.HasPrefix(cmd, "/") {
		log.Printf("max: unknown command=%q chatID=%d", cmd, upd.GetChatID())
		return
	}

	if upd.GetChatID() == m.Chats["Rogatka"] {
		m.saveRogatkaRequest(upd, text)
	}
}

func (m *MaxService) saveRogatkaRequest(upd *schemes.MessageCreatedUpdate, text string) {
	if upd.Message.Sender.IsBot {
		return
	}

	id, err := m.db.CreateRogatkaRequest(
		upd.GetChatID(),
		upd.Message.Sender.UserId,
		upd.Message.Sender.Username,
		upd.Message.Sender.Name,
		upd.Message.Body.Mid,
		text,
	)
	if err != nil {
		log.Printf("max rogatka: save error chatID=%d: %v", upd.GetChatID(), err)
		return
	}

	log.Printf("max rogatka: saved id=%d chatID=%d message=%q", id, upd.GetChatID(), text)
}

func (m *MaxService) handleGroupCommand(upd *schemes.MessageCreatedUpdate) {
	chatID := upd.GetChatID()
	ctx := context.Background()

	chat, err := m.Bot.Chats.GetChat(ctx, chatID)
	if err != nil {
		log.Printf("max /group: GetChat error chatID=%d: %v", chatID, err)
		return
	}

	logJSON("max /group: chat", chat)

	membership, err := m.Bot.Chats.GetChatMembership(ctx, chatID)
	if err != nil {
		log.Printf("max /group: GetChatMembership error chatID=%d: %v", chatID, err)
	} else {
		logJSON("max /group: membership", membership)
	}

	admins, err := m.Bot.Chats.GetChatAdmins(ctx, chatID)
	if err != nil {
		log.Printf("max /group: GetChatAdmins error chatID=%d: %v", chatID, err)
	} else {
		logJSON("max /group: admins", admins)
	}
}

func logJSON(prefix string, v any) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Printf("%s: %#v", prefix, v)
		return
	}
	log.Printf("%s:\n%s", prefix, string(data))
}
