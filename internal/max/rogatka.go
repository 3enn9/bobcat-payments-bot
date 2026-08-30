package max

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	maxbot "github.com/max-messenger/max-bot-api-client-go"
	"github.com/max-messenger/max-bot-api-client-go/schemes"
)

const rogatkaStatusSeparator = "\n---\n"

const (
	rogatkaStatusPending   = "❌ Не выполнен"
	rogatkaStatusCompleted = "✅ Выполнен"
)

func FormatDriverRequestText(driverName, message string) string {
	return fmt.Sprintf("%s: %s%s%s", driverName, message, rogatkaStatusSeparator, rogatkaStatusPending)
}

func FormatDriverRequestCompleted(driverName, message string) string {
	return fmt.Sprintf("%s: %s%s%s", driverName, message, rogatkaStatusSeparator, rogatkaStatusCompleted)
}

func (m *MaxService) SendDriverRequestNotification(driverName, message string) (string, error) {
	chatID, ok := m.Chats["DriverRequest"]
	if !ok {
		return "", fmt.Errorf("group name does not exist: DriverRequest")
	}

	ctx := context.Background()
	msg := maxbot.NewMessage().
		SetChat(chatID).
		SetText(FormatDriverRequestText(driverName, message))

	sent, err := m.Bot.Messages.SendWithResult(ctx, msg)
	if err != nil {
		return "", fmt.Errorf("send driver request: %w", err)
	}
	if sent == nil || sent.Body.Mid == "" {
		return "", errors.New("empty message id from MAX")
	}

	return sent.Body.Mid, nil
}

func (m *MaxService) MarkDriverRequestCompleted(messageID, driverName, message string) error {
	messageID = strings.TrimSpace(messageID)
	if messageID == "" {
		return nil
	}

	ctx := context.Background()
	editMsg := maxbot.NewMessage().SetText(FormatDriverRequestCompleted(driverName, message))
	if err := m.Bot.Messages.EditMessage(ctx, messageID, editMsg); err != nil {
		return fmt.Errorf("edit driver request message: %w", err)
	}

	return nil
}

func (m *MaxService) saveRogatkaRequest(upd *schemes.MessageCreatedUpdate, text string) {
	if upd.Message.Sender.IsBot {
		return
	}

	m.upsertRogatkaRequest(
		upd.GetChatID(),
		upd.Message.Sender.UserId,
		upd.Message.Sender.Username,
		upd.Message.Sender.Name,
		upd.Message.Body.Mid,
		text,
		"created",
	)
}

func (m *MaxService) handleRogatkaMessageEdited(upd *schemes.MessageEditedUpdate) {
	if upd.GetChatID() != m.Chats["Rogatka"] {
		return
	}
	if upd.Message.Sender.IsBot {
		return
	}

	text := strings.TrimSpace(upd.Message.Body.Text)
	if text == "" {
		return
	}

	m.upsertRogatkaRequest(
		upd.GetChatID(),
		upd.Message.Sender.UserId,
		upd.Message.Sender.Username,
		upd.Message.Sender.Name,
		upd.Message.Body.Mid,
		text,
		"edited",
	)
}

func (m *MaxService) upsertRogatkaRequest(
	chatID, userID int64,
	username, userName, messageID, message, action string,
) {
	id, created, err := m.db.SaveRogatkaRequest(
		chatID,
		userID,
		username,
		userName,
		messageID,
		message,
	)
	if err != nil {
		log.Printf("max rogatka: save error chatID=%d messageID=%s: %v", chatID, messageID, err)
		return
	}

	if created {
		log.Printf("max rogatka: created id=%d chatID=%d message=%q", id, chatID, message)
		return
	}
	log.Printf("max rogatka: updated id=%d chatID=%d action=%s message=%q", id, chatID, action, message)
}