package max

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	maxbot "github.com/max-messenger/max-bot-api-client-go"
	"github.com/max-messenger/max-bot-api-client-go/schemes"
)

const daysOffTomorrowButton = "Завтра"

type replyKeyboardButton struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type replyKeyboardAttachment struct {
	Type    string                `json:"type"`
	Buttons [][]replyKeyboardButton `json:"buttons"`
}

func daysOffReplyKeyboardAttachment() replyKeyboardAttachment {
	return replyKeyboardAttachment{
		Type: "reply_keyboard",
		Buttons: [][]replyKeyboardButton{
			{{Type: "message", Text: daysOffTomorrowButton}},
		},
	}
}

func (m *MaxService) sendChatMessageWithAttachments(ctx context.Context, chatID int64, text string, attachments ...interface{}) error {
	body := schemes.NewMessageBody{
		Text:        text,
		Attachments: attachments,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}

	u, err := url.Parse(maxbot.DefaultAPIURL + "messages")
	if err != nil {
		return err
	}

	q := u.Query()
	q.Set("access_token", m.token)
	q.Set("chat_id", strconv.FormatInt(chatID, 10))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("max send message: status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (m *MaxService) SetupDaysOffReplyKeyboard() error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	text := "Кнопка «Завтра» — список подтверждённых выходных на следующий день."
	return m.sendChatMessageWithAttachments(ctx, daysOffGroupID, text, daysOffReplyKeyboardAttachment())
}
