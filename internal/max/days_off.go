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

func (m *MaxService) RegisterBotCommands() error {
	ctx := context.Background()

	bot, err := m.Bot.Bots.GetBot(ctx)
	if err != nil {
		return fmt.Errorf("get bot info: %w", err)
	}

	commands := bot.Commands
	for _, c := range commands {
		if c.Name == "zavtra" {
			return nil
		}
	}

	commands = append(commands, schemes.BotCommand{
		Name:        "zavtra",
		Description: "Завтра",
	})

	_, err = m.Bot.Bots.PatchBot(ctx, &schemes.BotPatch{Commands: commands})
	if err != nil {
		return fmt.Errorf("patch bot commands: %w", err)
	}

	return nil
}

func (m *MaxService) handleDaysOffTomorrow(chatID int64) {
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	items, err := m.db.ListApprovedWorkerDaysOffOnDate(tomorrow)
	if err != nil {
		log.Printf("days off tomorrow: load date=%s: %v", tomorrow, err)
		_ = m.SendMessageInGroupID(chatID, "Не удалось загрузить список выходных.")
		return
	}

	text := formatTomorrowDaysOffMessage(tomorrow, items)
	if err := m.SendMessageInGroupID(chatID, text); err != nil {
		log.Printf("days off tomorrow: send chatID=%d: %v", chatID, err)
	}
}

func formatTomorrowDaysOffMessage(dateISO string, items []db.WorkerDaysOff) string {
	dateLabel := db.FormatDaysOffPeriod(dateISO, dateISO)
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Выходные на %s\n\n", dateLabel))

	if len(items) == 0 {
		b.WriteString("Никого нет.")
		return b.String()
	}

	for _, item := range items {
		line := item.WorkerName
		if item.DateFrom != dateISO || item.DateTo != dateISO {
			line += " (" + db.FormatDaysOffPeriod(item.DateFrom, item.DateTo) + ")"
		}
		b.WriteString("• " + line + "\n")
	}

	return strings.TrimSpace(b.String())
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

func (m *MaxService) answerCallback(upd *schemes.MessageCallbackUpdate, notification string) {
	ctx := context.Background()
	_, err := m.Bot.Messages.AnswerOnCallback(ctx, upd.Callback.CallbackID, &schemes.CallbackAnswer{
		Notification: notification,
	})
	if err != nil {
		log.Printf("days off callback: answer: %v", err)
	}
}
