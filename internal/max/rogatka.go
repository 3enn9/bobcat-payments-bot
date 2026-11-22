package max

import (
	"context"
	"errors"
	"fmt"
	"strings"

	maxbot "github.com/max-messenger/max-bot-api-client-go"
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
