package max

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"PaymentsBot/internal/db"

	maxbot "github.com/max-messenger/max-bot-api-client-go"
)

const daysOffTomorrowButton = "Завтра"

func daysOffTomorrowKeyboard() *maxbot.Keyboard {
	kb := maxbot.InlineKeyboard(
		maxbot.Row(maxbot.BtnMsg(daysOffTomorrowButton)),
	)
	return kb
}

func (m *MaxService) sendDaysOffGroupMessage(text string, withKeyboard bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	msg := maxbot.NewMessage().SetChat(daysOffGroupID).SetText(text)
	if withKeyboard {
		msg.AddKeyboard(daysOffTomorrowKeyboard())
	}

	return m.Bot.Messages.Send(ctx, msg)
}

func (m *MaxService) SetupDaysOffReplyKeyboard() error {
	text := "Кнопка «Завтра» — список подтверждённых выходных на следующий день."
	if err := m.sendDaysOffGroupMessage(text, true); err != nil {
		return fmt.Errorf("setup days off keyboard: %w", err)
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
	if err := m.sendDaysOffGroupMessage(text, true); err != nil {
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
