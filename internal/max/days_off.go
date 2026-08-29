package max

import (
	"PaymentsBot/internal/db"
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	maxbot "github.com/max-messenger/max-bot-api-client-go"
	"github.com/max-messenger/max-bot-api-client-go/schemes"
)

const daysOffGroupID int64 = -78302034737888

func (m *MaxService) SendDaysOffApproval(item db.WorkerDaysOff, id int64) (string, error) {
	text := db.FormatDaysOffMessage(item)

	keyboard := m.Bot.Messages.NewKeyboardBuilder()
	keyboard.AddRow().
		AddCallback("Подтвердить", schemes.POSITIVE, fmt.Sprintf("dayoff:approve:%d", id)).
		AddCallback("Отменить", schemes.NEGATIVE, fmt.Sprintf("dayoff:reject:%d", id))

	ctx := context.Background()
	msg := maxbot.NewMessage().
		SetChat(daysOffGroupID).
		SetText(text).
		AddKeyboard(keyboard)

	sent, err := m.Bot.Messages.SendWithResult(ctx, msg)
	if err != nil {
		return "", fmt.Errorf("send days off message: %w", err)
	}
	if sent == nil || sent.Body.Mid == "" {
		return "", errors.New("empty message id from MAX")
	}
	return sent.Body.Mid, nil
}

func (m *MaxService) handleDaysOffCallback(upd *schemes.MessageCallbackUpdate) {
	payload := strings.TrimSpace(upd.Callback.Payload)
	parts := strings.Split(payload, ":")
	if len(parts) != 3 || parts[0] != "dayoff" {
		return
	}

	action := parts[1]
	id, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || id <= 0 {
		return
	}

	status := ""
	switch action {
	case "approve":
		status = "approved"
	case "reject":
		status = "rejected"
	default:
		return
	}

	deciderName := strings.TrimSpace(upd.Callback.User.Name)
	if deciderName == "" {
		deciderName = strings.TrimSpace(upd.Callback.User.Username)
	}
	if deciderName == "" {
		deciderName = "Руководитель"
	}

	record, err := m.db.GetWorkerDaysOffForDecision(id)
	if err != nil {
		log.Printf("days off callback: load id=%d: %v", id, err)
		m.answerCallback(upd, "Заявка не найдена")
		return
	}

	if record.Status != "pending" {
		m.answerCallback(upd, "Решение уже принято")
		return
	}

	if err := m.db.DecideWorkerDaysOff(id, status, deciderName, upd.Callback.User.UserId); err != nil {
		if errors.Is(err, db.ErrDaysOffAlreadyDecided) {
			m.answerCallback(upd, "Решение уже принято")
			return
		}
		log.Printf("days off callback: decide id=%d: %v", id, err)
		m.answerCallback(upd, "Ошибка сохранения")
		return
	}

	original := record.OriginalText
	if upd.Message != nil && strings.TrimSpace(upd.Message.Body.Text) != "" {
		original = strings.TrimSpace(upd.Message.Body.Text)
		// strip previous decision footer if re-clicked
		if idx := strings.Index(original, "\n---\n"); idx >= 0 {
			original = strings.TrimSpace(original[:idx])
		}
	}

	decisionLine := formatDaysOffDecision(status, deciderName, time.Now())
	newText := original + "\n---\n" + decisionLine

	messageID := record.MaxMessageID
	if upd.Message != nil && upd.Message.Body.Mid != "" {
		messageID = upd.Message.Body.Mid
	}

	if messageID != "" {
		ctx := context.Background()
		editMsg := maxbot.NewMessage().SetText(newText)
		if err := m.Bot.Messages.EditMessage(ctx, messageID, editMsg); err != nil {
			log.Printf("days off callback: edit message id=%d: %v", id, err)
		}
	}

	m.answerCallback(upd, decisionLine)
}

func (m *MaxService) answerCallback(upd *schemes.MessageCallbackUpdate, notification string) {
	ctx := context.Background()
	_, err := m.Bot.Messages.AnswerOnCallback(ctx, upd.Callback.CallbackID, &schemes.CallbackAnswer{
		Notification: notification,
	})
	if err != nil {
		log.Printf("days off callback: answer: %v", err)
	}
}

func formatDaysOffDecision(status, name string, at time.Time) string {
	when := at.Format("02.01.2006 15:04")
	switch status {
	case "approved":
		return fmt.Sprintf("✅ Подтверждено: %s · %s", name, when)
	default:
		return fmt.Sprintf("❌ Отклонено: %s · %s", name, when)
	}
}
