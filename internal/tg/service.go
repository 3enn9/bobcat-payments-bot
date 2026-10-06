package tg

import (
	"PaymentsBot/internal/domain/messenger"
	"PaymentsBot/internal/payments"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TelegramService struct {
	bot            *tgbotapi.BotAPI
	Chats          map[string]int64
	payments       *payments.PaymentsService
	invoicesChatID int64
}

func NewTelegramService(token string, payments *payments.PaymentsService, invoicesChatID int64) (*TelegramService, error) {
	bot, err := tgbotapi.NewBotAPI(token)

	if err != nil {
		return nil, err
	}

	chats := map[string]int64{
		"Payments": -1003380906513,
		"Fuels":    -1003368403742,
		"Cash":     -1003797529492,
	}

	return &TelegramService{bot: bot, Chats: chats, payments: payments, invoicesChatID: invoicesChatID}, nil
}

// SendInvoicePages sends rendered invoice pages to the Telegram forum topic
// selected by the legacy sender's filename routing.
func (s *TelegramService) SendInvoicePages(filename, caption string, pages [][]byte) error {
	if s.invoicesChatID == 0 {
		return fmt.Errorf("CHAT_ID is not configured")
	}
	topicID, ok := invoiceTopic(filename)
	if !ok {
		return fmt.Errorf("unknown invoice topic for %q", filename)
	}
	for i, page := range pages {
		for {
			retryAfter, err := s.sendInvoicePhoto(topicID, i, caption, page)
			if err == nil {
				break
			}
			if retryAfter > 0 {
				time.Sleep(time.Duration(retryAfter+1) * time.Second)
				continue
			}
			return fmt.Errorf("send invoice page %d: %w", i+1, err)
		}
		time.Sleep(time.Second)
	}
	return nil
}

func (s *TelegramService) sendInvoicePhoto(topicID, pageIndex int, caption string, image []byte) (int, error) {
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	_ = form.WriteField("chat_id", strconv.FormatInt(s.invoicesChatID, 10))
	_ = form.WriteField("message_thread_id", strconv.Itoa(topicID))
	if pageIndex == 0 && caption != "" {
		_ = form.WriteField("caption", caption)
	}
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="photo"; filename="page_%d.png"`, pageIndex+1))
	header.Set("Content-Type", "image/png")
	part, err := form.CreatePart(header)
	if err != nil {
		return 0, err
	}
	if _, err := part.Write(image); err != nil {
		return 0, err
	}
	if err := form.Close(); err != nil {
		return 0, err
	}

	request, err := http.NewRequest(http.MethodPost, "https://api.telegram.org/bot"+s.bot.Token+"/sendPhoto", &body)
	if err != nil {
		return 0, err
	}
	request.Header.Set("Content-Type", form.FormDataContentType())
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	var result struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
		Parameters  struct {
			RetryAfter int `json:"retry_after"`
		} `json:"parameters"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&result); err != nil {
		return 0, err
	}
	if !result.OK {
		return result.Parameters.RetryAfter, fmt.Errorf("telegram sendPhoto: %s", result.Description)
	}
	return 0, nil
}

func invoiceTopic(filename string) (int, bool) {
	name := strings.ToLower(filepath.Base(filename))
	for _, route := range []struct {
		marker string
		topic  int
	}{
		{"акт сверки", 1657}, {"сст", 1221}, {"анн", 1223},
		{"анв", 1227}, {"адн", 1233}, {"сан", 1237},
	} {
		if strings.Contains(name, route.marker) {
			return route.topic, true
		}
	}
	return 0, false
}

func (s *TelegramService) SendMessageInGroupID(chatID int64, message string) error {

	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = "HTML"

	sentMsg, err := s.bot.Send(msg)
	if err != nil {
		return messenger.ErrSendMessage
	}

	if strings.Contains(message, "Касса:") {
		pin := tgbotapi.PinChatMessageConfig{ChatID: chatID, MessageID: sentMsg.MessageID, DisableNotification: true}
		resp, err := s.bot.Request(pin)
		if err != nil {
			fmt.Printf("Error pin message")
			return nil
		}
		fmt.Println(resp.Description)

	}
	return nil
}

func (s *TelegramService) SendMessageInGroupName(nameGroup string, message string) error {
	chatID, ok := s.Chats[nameGroup]
	if !ok {
		return fmt.Errorf("groupNmae not in map chats")
	}
	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = "HTML"

	_, err := s.bot.Send(msg)
	if err != nil {
		fmt.Printf("Error send message")
		return nil
	}

	return nil
}

func (s *TelegramService) Updates(u tgbotapi.Update) error {
	if u.Message == nil {
		return messenger.ErrMessageEmpty
	}
	text := u.Message.Text
	chatID := u.Message.Chat.ID
	chatName := u.Message.Chat.Title

	if strings.HasPrefix(text, "/start") {
		user := u.Message.From
		if user != nil {
			name := strings.TrimSpace(strings.TrimSpace(user.FirstName + " " + user.LastName))
			log.Printf("bot open: source=telegram user_id=%d name=%q username=%q chat_id=%d", user.ID, name, user.UserName, chatID)
		}
	}

	switch {
	case strings.HasPrefix(text, "/add "):
		cashCh := make(chan string, 1)
		result, err := s.payments.AddPayment(cashCh, chatID, text, chatName)

		if err != nil {
			log.Println("error AddPayments")
			return nil
		}
		err = s.SendMessageInGroupID(chatID, result.GroupMessage)
		if err != nil {
			log.Printf("error send message in groupID %v", err)
		}
		err = s.SendMessageInGroupName("Cash", result.CashMessage)
		if err != nil {
			log.Printf("error send message in groupName %v", err)
		}
	case strings.HasPrefix(text, "/all"):
		msg, err := s.payments.AllBalance(chatID)
		if err != nil {
			log.Printf("error AllBalance %v", err)
			return nil
		}
		err = s.SendMessageInGroupID(chatID, msg)
		if err != nil {
			log.Printf("error send message in groupID %v", err)
		}
	case strings.HasPrefix(text, "/dep "):
		err := s.payments.Deposit(chatID, text, chatName)
		if err != nil {
			log.Printf("error deposit %v", err)
		}
	case strings.HasPrefix(text, "/salary "):
		err := s.payments.Salary(chatID)
		if err != nil {
			log.Printf("error salary %v", err)
		}
	}

	return nil
}
