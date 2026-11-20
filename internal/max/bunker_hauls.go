package max

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/max-messenger/max-bot-api-client-go/schemes"
)

func (m *MaxService) saveBunkerHaulMessage(upd *schemes.MessageCreatedUpdate) error {
	message := upd.Message
	if message.Sender.IsBot || strings.TrimSpace(message.Body.Text) == "" {
		return nil
	}

	messageDate := time.Now()
	if message.Timestamp > 0 {
		messageDate = time.UnixMilli(message.Timestamp).In(time.Local)
	}

	senderName := strings.TrimSpace(message.Sender.Name)
	if senderName == "" {
		senderName = strings.TrimSpace(strings.TrimSpace(message.Sender.FirstName) + " " + strings.TrimSpace(message.Sender.LastName))
	}
	if senderName == "" {
		senderName = strings.TrimSpace(message.Sender.Username)
	}

	if err := m.db.SaveBunkerHaulMessage(
		upd.GetChatID(),
		message.Body.Mid,
		messageDate,
		message.Sender.UserId,
		senderName,
		strings.TrimSpace(message.Sender.Username),
		message.Body.Text,
	); err != nil {
		log.Printf("max bunker_hauls: save message mid=%s: %v", message.Body.Mid, err)
		return fmt.Errorf("save bunker haul message: %w", err)
	}
	return nil
}
