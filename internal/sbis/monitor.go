package sbis

import (
	"PaymentsBot/internal/db"
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	maxbot "github.com/max-messenger/max-bot-api-client-go"
)

const (
	endpoint   = "https://online.sbis.ru"
	maxChatID  = int64(-80188244966112)
	checkEvery = time.Hour
	// Recover recent events after a cursor reset or a period when polling stalled.
	bootstrap = 7 * 24 * time.Hour
	pageSize  = "50"
)

type Monitor struct {
	login, password, account string
	db                       *db.Database
	max                      *maxbot.Api
	client                   *http.Client
	session                  string
	lastMessageSent          time.Time
}

func NewMonitor(login, password, account string, database *db.Database, bot *maxbot.Api) *Monitor {
	return &Monitor{login: login, password: password, account: account, db: database, max: bot, client: &http.Client{Timeout: 30 * time.Second}}
}

func (m *Monitor) Run(ctx context.Context) {
	if strings.TrimSpace(m.login) == "" || strings.TrimSpace(m.password) == "" {
		log.Println("SBIS: SBIS_LOGIN/SBIS_PASSWORD не заданы, мониторинг отключен")
		return
	}
	// Catch recent changes on startup; the event table prevents duplicate notifications.
	if err := m.poll(ctx); err != nil {
		log.Printf("SBIS poll: %v", err)
	}
	ticker := time.NewTicker(checkEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := m.poll(ctx); err != nil {
				log.Printf("SBIS poll: %v", err)
			}
		}
	}
}

func (m *Monitor) poll(ctx context.Context) error {
	if m.session == "" {
		params := map[string]any{"Логин": m.login, "Пароль": m.password}
		if m.account != "" {
			params["НомерАккаунта"] = m.account
		}
		var sid string
		if err := m.call(ctx, "/auth/service/", "СБИС.Аутентифицировать", map[string]any{"Параметр": params}, "", &sid); err != nil {
			return fmt.Errorf("authenticate: %w", err)
		}
		m.session = sid
	}

	cursor, err := m.loadCursor(ctx)
	if err != nil {
		return fmt.Errorf("load cursor: %w", err)
	}
	fromTime := time.Now().Add(-bootstrap)
	if cursor.EventID != "" && cursor.DateTime != "" {
		if cursorTime, err := time.ParseInLocation("02.01.2006 15.04.05", cursor.DateTime, time.Local); err == nil {
			// Start each poll from the last event time with a small overlap.
			// Event IDs are only for fetching subsequent pages within this poll.
			fromTime = cursorTime.Add(-time.Minute)
		}
	}
	queryFrom := fromTime.Format("02.01.2006 15.04.05")
	filter := map[string]any{
		"ДатаВремяС":         queryFrom,
		"ПолныйСертификатЭП": "Нет",
		"Навигация":          map[string]any{"РазмерСтраницы": pageSize},
	}
	var scannedDocs, sent, pages int
	advanced := false
	lastCursor := ""
	for {
		var response struct {
			Document   []document `json:"Документ"`
			Navigation struct {
				HasMore string `json:"ЕстьЕще"`
			} `json:"Навигация"`
		}
		if err := m.call(ctx, "/service/?srv=1", "СБИС.СписокИзменений", map[string]any{"Фильтр": filter}, m.session, &response); err != nil {
			m.session = ""
			return err
		}
		pages++
		scannedDocs += len(response.Document)
		if len(response.Document) == 0 {
			break
		}

		for _, doc := range response.Document {
			if !strings.EqualFold(doc.Direction, "Исходящий") {
				continue
			}
			for _, event := range doc.Events {
				if event.ID == "" || !eventHasIncomingSignature(event) {
					continue
				}
				key := eventKey(doc.ID, event.ID)
				exists, err := m.wasNotified(ctx, key)
				if err != nil {
					return err
				}
				if exists {
					continue
				}
				text := formatMessage(doc, event)
				msg := maxbot.NewMessage().SetChat(maxChatID).SetText(text)
				if err := m.sendMessage(ctx, msg); err != nil {
					return fmt.Errorf("MAX send document %s: %w", doc.ID, err)
				}
				if _, err := m.db.DB.ExecContext(ctx, `INSERT IGNORE INTO sbis_edo_notified (event_key, document_id, document_number, counterparty, signed_at) VALUES (?, ?, ?, ?, ?)`, key, doc.ID, doc.Number, doc.Counterparty.name(), parseDateTime(event.DateTime)); err != nil {
					return fmt.Errorf("save notification: %w", err)
				}
				sent++
			}
		}

		lastDoc, lastEvent := lastDocumentEvent(response.Document)
		if lastDoc == nil || lastEvent == nil {
			return fmt.Errorf("СБИС returned a page without a usable event cursor")
		}
		cursorKey := eventKey(lastDoc.ID, lastEvent.ID)
		if cursorKey == lastCursor {
			return fmt.Errorf("СБИС pagination did not advance past event %s", lastEvent.ID)
		}
		lastCursor = cursorKey
		cursor = eventCursor{DateTime: lastEvent.DateTime, EventID: lastEvent.ID, DocumentID: lastDoc.ID}
		advanced = true
		if strings.EqualFold(response.Navigation.HasMore, "Нет") {
			break
		}
		// СБИС.СписокИзменений continues from the last event, rather than a page number.
		filter["ИдентификаторСобытия"] = lastEvent.ID
		filter["ИдентификаторДокумента"] = lastDoc.ID
		filter["ДатаВремяС"] = lastEvent.DateTime
	}
	if !advanced {
		// СБИС по умолчанию не возвращает события за последнюю минуту.
		// Сохраняем небольшой перекрывающийся интервал для следующего опроса.
		cursor = eventCursor{DateTime: time.Now().Add(-time.Minute).Format("02.01.2006 15.04.05")}
	}
	if err := m.saveCursor(ctx, cursor); err != nil {
		return fmt.Errorf("save cursor: %w", err)
	}
	log.Printf("SBIS poll: from=%s scanned=%d pages=%d notifications_sent=%d", queryFrom, scannedDocs, pages, sent)
	return nil
}

func (m *Monitor) sendMessage(ctx context.Context, msg *maxbot.Message) error {
	const minInterval = 600 * time.Millisecond
	if !m.lastMessageSent.IsZero() {
		if wait := minInterval - time.Since(m.lastMessageSent); wait > 0 {
			timer := time.NewTimer(wait)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
			}
		}
	}

	backoff := time.Second
	for {
		if err := m.max.Messages.Send(ctx, msg); err != nil {
			var apiErr *maxbot.APIError
			if !errors.As(err, &apiErr) || apiErr.Code != http.StatusTooManyRequests {
				return err
			}

			log.Printf("SBIS MAX rate limit (429); retrying in %s", backoff)
			timer := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
			backoff *= 2
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
			continue
		}
		m.lastMessageSent = time.Now()
		return nil
	}
}

type eventCursor struct {
	DateTime   string
	EventID    string
	DocumentID string
}

func (m *Monitor) loadCursor(ctx context.Context) (eventCursor, error) {
	var cursor eventCursor
	err := m.db.DB.QueryRowContext(ctx, `SELECT event_datetime, event_id, document_id FROM sbis_edo_cursor WHERE cursor_id = 1`).Scan(&cursor.DateTime, &cursor.EventID, &cursor.DocumentID)
	if err == sql.ErrNoRows {
		return eventCursor{DateTime: time.Now().Add(-bootstrap).Format("02.01.2006 15.04.05")}, nil
	}
	return cursor, err
}

func (m *Monitor) saveCursor(ctx context.Context, cursor eventCursor) error {
	_, err := m.db.DB.ExecContext(ctx, `INSERT INTO sbis_edo_cursor (cursor_id, event_datetime, event_id, document_id) VALUES (1, ?, ?, ?) ON DUPLICATE KEY UPDATE event_datetime = VALUES(event_datetime), event_id = VALUES(event_id), document_id = VALUES(document_id)`, cursor.DateTime, cursor.EventID, cursor.DocumentID)
	return err
}

func lastDocumentEvent(docs []document) (*document, *event) {
	for i := len(docs) - 1; i >= 0; i-- {
		for j := len(docs[i].Events) - 1; j >= 0; j-- {
			if docs[i].Events[j].ID != "" {
				return &docs[i], &docs[i].Events[j]
			}
		}
	}
	return nil, nil
}

func (m *Monitor) call(ctx context.Context, path, method string, params any, session string, dst any) error {
	requestBody := map[string]any{"jsonrpc": "2.0", "method": method, "params": params, "id": 0}
	b, err := json.Marshal(requestBody)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json-rpc;charset=utf-8")
	req.Header.Set("User-Agent", "PaymentsBot/1.0")
	if session != "" {
		req.Header.Set("X-SBISSessionID", session)
	}
	resp, err := m.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("СБИС вернул HTTP 401")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("СБИС HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	var envelope struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return fmt.Errorf("decode СБИС response: %w", err)
	}
	if envelope.Error != nil {
		return fmt.Errorf("СБИС %s: %s", method, envelope.Error.Message)
	}
	if err := json.Unmarshal(envelope.Result, dst); err != nil {
		return fmt.Errorf("decode СБИС %s result: %w", method, err)
	}
	return nil
}

type document struct {
	ID           string  `json:"Идентификатор"`
	Number       string  `json:"Номер"`
	Title        string  `json:"Название"`
	Direction    string  `json:"Направление"`
	Counterparty party   `json:"Контрагент"`
	Events       []event `json:"Событие"`
}
type party struct {
	Legal struct {
		Name string `json:"Название"`
	} `json:"СвЮЛ"`
	Person struct {
		Family string `json:"Фамилия"`
		Name   string `json:"Имя"`
		Middle string `json:"Отчество"`
	} `json:"СвФЛ"`
}

func (p party) name() string {
	if p.Legal.Name != "" {
		return p.Legal.Name
	}
	return strings.TrimSpace(p.Person.Family + " " + p.Person.Name + " " + p.Person.Middle)
}

type event struct {
	ID          string       `json:"Идентификатор"`
	DateTime    string       `json:"ДатаВремя"`
	Name        string       `json:"Название"`
	Attachments []attachment `json:"Вложение"`
}
type attachment struct {
	Name       string      `json:"Название"`
	Direction  string      `json:"Направление"`
	Signatures []signature `json:"Подпись"`
}
type signature struct {
	Direction   string `json:"Направление"`
	Certificate struct {
		Name    string `json:"ФИО"`
		Company string `json:"Название"`
	} `json:"Сертификат"`
}

func eventHasIncomingSignature(e event) bool {
	for _, a := range e.Attachments {
		for _, s := range a.Signatures {
			if strings.EqualFold(s.Direction, "Входящая") || strings.EqualFold(s.Direction, "Входящий") {
				return true
			}
			// Some API versions omit signature direction in the event payload. A signature
			// attached to an incoming service attachment still signals the counterparty action.
			if s.Certificate.Name != "" && strings.EqualFold(a.Direction, "Входящий") {
				return true
			}
		}
	}
	return false
}
func eventKey(docID, eventID string) string {
	h := sha256.Sum256([]byte(docID + "/" + eventID))
	return hex.EncodeToString(h[:])
}
func (m *Monitor) wasNotified(ctx context.Context, key string) (bool, error) {
	var n int
	err := m.db.DB.QueryRowContext(ctx, `SELECT 1 FROM sbis_edo_notified WHERE event_key = ? LIMIT 1`, key).Scan(&n)
	if err == nil {
		return true, nil
	}
	if err == sql.ErrNoRows {
		return false, nil
	}
	return false, err
}
func formatMessage(d document, e event) string {
	parts := []string{"✅ Контрагент подписал документ в СБИС ЭДО"}
	if d.Counterparty.name() != "" {
		parts = append(parts, "Контрагент: "+d.Counterparty.name())
	}
	if d.Title != "" {
		parts = append(parts, "Документ: "+d.Title)
	} else if d.Number != "" {
		parts = append(parts, "Документ № "+d.Number)
	}
	if e.DateTime != "" {
		parts = append(parts, "Время: "+e.DateTime)
	}
	for _, a := range e.Attachments {
		for _, s := range a.Signatures {
			if s.Certificate.Name != "" {
				parts = append(parts, "Подписант: "+s.Certificate.Name)
				return strings.Join(parts, "\n")
			}
		}
	}
	return strings.Join(parts, "\n")
}
func parseDateTime(s string) any {
	if t, err := time.ParseInLocation("02.01.2006 15.04.05", s, time.Local); err == nil {
		return t
	}
	return nil
}
