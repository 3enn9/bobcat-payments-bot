package max

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	maxbot "github.com/max-messenger/max-bot-api-client-go"
	"github.com/max-messenger/max-bot-api-client-go/schemes"
)

const reconciliationVerifiedPayloadPrefix = "reconciliation:verified:"

func (m *MaxService) SendReconciliationPhotos(fileName, text string, photos []PhotoUpload) error {
	chatID, ok := m.Chats["Invoices"]
	if !ok {
		return fmt.Errorf("group name does not exist: Invoices")
	}
	return m.sendPhotosToChat(chatID, text, photos, m.reconciliationKeyboard(fileName))
}

func (m *MaxService) SendReconciliationFile(fileName string, reader io.Reader) error {
	chatID, ok := m.Chats["Invoices"]
	if !ok {
		return fmt.Errorf("group name does not exist: Invoices")
	}
	info, err := m.Bot.Uploads.UploadMediaFromReaderWithName(context.Background(), schemes.FILE, reader, fileName)
	if err != nil {
		return fmt.Errorf("upload reconciliation file: %w", err)
	}
	msg := maxbot.NewMessage().SetChat(chatID).AddFile(info).AddKeyboard(m.reconciliationKeyboard(fileName))
	if err := m.Bot.Messages.Send(context.Background(), msg); err != nil {
		return fmt.Errorf("send reconciliation file: %w", err)
	}
	return nil
}

func (m *MaxService) reconciliationKeyboard(fileName string) *maxbot.Keyboard {
	payload := reconciliationVerifiedPayloadPrefix + base64.RawURLEncoding.EncodeToString([]byte(strings.TrimSpace(fileName)))
	keyboard := m.Bot.Messages.NewKeyboardBuilder()
	keyboard.AddRow().AddCallback("Отправить", schemes.POSITIVE, payload)
	return keyboard
}

func (m *MaxService) handleReconciliationCallback(upd *schemes.MessageCallbackUpdate) {
	if upd.Message == nil || upd.GetChatID() != m.Chats["Invoices"] {
		m.answerCallback(upd, "Кнопка доступна только в группе счетов")
		return
	}

	payload := strings.TrimPrefix(upd.Callback.Payload, reconciliationVerifiedPayloadPrefix)
	fileNameBytes, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		m.answerCallback(upd, "Не удалось прочитать название документа")
		return
	}
	fileName := strings.TrimSpace(string(fileNameBytes))
	if fileName == "" || strings.ContainsAny(fileName, "\r\n") {
		m.answerCallback(upd, "Некорректное название документа")
		return
	}

	officeID, ok := m.Chats["Office"]
	if !ok {
		m.answerCallback(upd, "Не настроена группа Офис")
		return
	}
	if err := m.SendMessageInGroupID(officeID, fileName+" - верный"); err != nil {
		m.answerCallback(upd, "Не удалось отправить в группу Офис")
		return
	}
	m.answerCallback(upd, "Отправлено в группу Офис")
}
