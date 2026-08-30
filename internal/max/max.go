package max

import (
	"PaymentsBot/internal/db"
	"PaymentsBot/internal/payments"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
		"Payments":      -77028763384544,
		"Fuels":         -77028768299744,
		"Cash":          -77028778785504,
		"Rogatka":       -71392114984255,
		"DriverRequest": -78173743561440,
		"WorkerDone":    -78179579607776,
		"Invoices":      -78218659838688,
		"DaysOff":       -78302034737888,
		"Garage":        -78319159032544,
	}
	return &MaxService{Bot: api, Chats: chats, payments: payments, db: database}, nil
}

type PhotoUpload struct {
	Name   string
	Reader io.Reader
}

func (m *MaxService) SendMessageWithPhotos(groupName, text string, photos []PhotoUpload) error {
	chatID, ok := m.Chats[groupName]
	if !ok {
		return fmt.Errorf("group name does not exist: %s", groupName)
	}

	ctx := context.Background()
	msg := maxbot.NewMessage().SetChat(chatID).SetText(text)

	for _, photo := range photos {
		tokens, err := m.Bot.Uploads.UploadPhotoFromReaderWithName(ctx, photo.Reader, photo.Name)
		if err != nil {
			return fmt.Errorf("upload photo %s: %w", photo.Name, err)
		}
		msg.AddPhoto(tokens)
	}

	if err := m.Bot.Messages.Send(ctx, msg); err != nil {
		return fmt.Errorf("send message: %w", err)
	}

	return nil
}

func (m *MaxService) SendFileToGroup(groupName, fileName string, reader io.Reader) error {
	chatID, ok := m.Chats[groupName]
	if !ok {
		return fmt.Errorf("group name does not exist: %s", groupName)
	}

	ctx := context.Background()
	info, err := m.Bot.Uploads.UploadMediaFromReaderWithName(ctx, schemes.FILE, reader, fileName)
	if err != nil {
		return fmt.Errorf("upload file: %w", err)
	}

	msg := maxbot.NewMessage().SetChat(chatID).AddFile(info)
	if err := m.Bot.Messages.Send(ctx, msg); err != nil {
		return fmt.Errorf("send file message: %w", err)
	}
	return nil
}

// SendFileAndPhotosToGroup отправляет PDF и фото отдельными сообщениями (MAX — один файл на сообщение).
func (m *MaxService) SendFileAndPhotosToGroup(groupName, fileName string, reader io.Reader, photos []PhotoUpload) error {
	if err := m.SendFileToGroup(groupName, fileName, reader); err != nil {
		return err
	}
	if len(photos) == 0 {
		return nil
	}
	return m.SendMessageWithPhotos(groupName, "", photos)
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
		return fmt.Errorf("group name does not exist: %s", nameGroup)
	}

	msg := maxbot.NewMessage().SetChat(chatID).SetText(message)
	err := m.Bot.Messages.Send(context.Background(), msg)
	if err != nil {
		log.Printf("error send message %v", err)
		return err
	}
	return nil
}

func (m *MaxService) Updates(update schemes.UpdateInterface) error {
	switch upd := update.(type) {

	case *schemes.MessageCreatedUpdate:
		m.handleMessage(upd)

	case *schemes.MessageCallbackUpdate:
		m.handleDaysOffCallback(upd)

	case *schemes.MessageEditedUpdate:
		m.handleRogatkaMessageEdited(upd)

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
	case "/zavtra":
		if !upd.Message.Sender.IsBot {
			m.handleDaysOffTomorrow(upd.GetChatID())
		}
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
