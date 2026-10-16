package sbis

import (
	"PaymentsBot/internal/db"
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
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
	checkEvery = time.Minute
)

type Monitor struct {
	login, password, account string
	db                       *db.Database
	max                      *maxbot.Api
	client                   *http.Client
	session                  string
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

	from := time.Now().Add(-10 * time.Minute).Format("02.01.2006 15.04.05")
	var response struct {
		Document []document `json:"Документ"`
	}
	filter := map[string]any{"ДатаВремяС": from, "ПолныйСертификатЭП": "Нет"}
	// The method takes one argument, Фильтр. Навигация belongs inside it;
	// placing it beside Фильтр makes JSON-RPC resolve the nonexistent /2 overload.
	filter["Навигация"] = map[string]any{"РазмерСтраницы": "50"}
	if err := m.call(ctx, "/service/?srv=1", "СБИС.СписокИзменений", map[string]any{"Фильтр": filter}, m.session, &response); err != nil {
		m.session = ""
		return err
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
			if err := m.max.Messages.Send(ctx, msg); err != nil {
				return fmt.Errorf("MAX send document %s: %w", doc.ID, err)
			}
			if _, err := m.db.DB.ExecContext(ctx, `INSERT IGNORE INTO sbis_edo_notified (event_key, document_id, document_number, counterparty, signed_at) VALUES (?, ?, ?, ?, ?)`, key, doc.ID, doc.Number, doc.Counterparty.name(), parseDateTime(event.DateTime)); err != nil {
				return fmt.Errorf("save notification: %w", err)
			}
		}
	}
	return nil
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
