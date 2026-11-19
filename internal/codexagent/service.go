package codexagent

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Service struct {
	DB         *sql.DB
	URL, Token string
	Owner      int64
	Send       func(int64, string) error
	mu         sync.Mutex
}
type Request struct {
	Prompt  string `json:"prompt"`
	Session string `json:"session"`
}
type Result struct {
	Text    string `json:"text"`
	Session string `json:"session"`
}

func (s *Service) Init(ctx context.Context) error {
	for _, q := range []string{
		`CREATE TABLE IF NOT EXISTS codex_jobs (id BIGINT AUTO_INCREMENT PRIMARY KEY, message_id VARCHAR(128) NOT NULL UNIQUE, chat_id BIGINT NOT NULL, prompt TEXT NOT NULL, response MEDIUMTEXT, sent_chars INT NOT NULL DEFAULT 0, done BOOLEAN NOT NULL DEFAULT FALSE, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS codex_dialog (chat_id BIGINT PRIMARY KEY, session_id VARCHAR(128) NOT NULL) ENGINE=InnoDB`,
	} {
		if _, err := s.DB.ExecContext(ctx, q); err != nil {
			return err
		}
	}
	return nil
}
func (s *Service) Enqueue(ctx context.Context, mid string, chat int64, prompt string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if mid == "" || len(mid) > 128 || len([]rune(prompt)) > 16000 {
		return fmt.Errorf("некорректное сообщение или слишком длинный вопрос")
	}
	var exists, n int
	if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM codex_jobs WHERE message_id=?`, mid).Scan(&exists); err != nil {
		return err
	}
	if exists > 0 {
		return nil
	}
	if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM codex_jobs WHERE done=FALSE`).Scan(&n); err != nil {
		return err
	}
	if n >= 10 {
		return fmt.Errorf("очередь агента заполнена, попробуй позже")
	}
	_, err := s.DB.ExecContext(ctx, `INSERT INTO codex_jobs(message_id,chat_id,prompt) VALUES(?,?,?)`, mid, chat, prompt)
	return err
}
func (s *Service) Run(ctx context.Context) {
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := s.step(ctx); err != nil {
				log.Printf("codex job: %v", err)
			}
		}
	}
}
func (s *Service) step(ctx context.Context) error {
	var id, chat int64
	var prompt string
	var response sql.NullString
	var sent int
	err := s.DB.QueryRowContext(ctx, `SELECT id,chat_id,prompt,response,sent_chars FROM codex_jobs WHERE done=FALSE ORDER BY id LIMIT 1`).Scan(&id, &chat, &prompt, &response, &sent)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	if !response.Valid {
		result := Result{}
		if prompt == "/new" {
			_, err = s.DB.ExecContext(ctx, `DELETE FROM codex_dialog WHERE chat_id=?`, chat)
			result.Text = "Начат новый разговор."
		} else {
			var session string
			err = s.DB.QueryRowContext(ctx, `SELECT session_id FROM codex_dialog WHERE chat_id=?`, chat).Scan(&session)
			if err == sql.ErrNoRows {
				err = nil
			}
			if err != nil {
				return err
			}
			result, err = s.ask(ctx, Request{prompt, session})
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			log.Printf("codex request failed job=%d: %v", id, err)
			result = Result{Text: "Агент сейчас недоступен: проверьте авторизацию Codex, лимит подписки или состояние сервиса. Попробуйте ещё раз позже."}
		}
		if strings.TrimSpace(result.Text) == "" {
			result.Text = "Агент не вернул ответ. Попробуй уточнить вопрос."
		}
		tx, e := s.DB.BeginTx(ctx, nil)
		if e != nil {
			return e
		}
		defer tx.Rollback()
		if result.Session != "" {
			_, e = tx.ExecContext(ctx, `INSERT INTO codex_dialog(chat_id,session_id) VALUES(?,?) ON DUPLICATE KEY UPDATE session_id=VALUES(session_id)`, chat, result.Session)
			if e != nil {
				return e
			}
		}
		if _, e = tx.ExecContext(ctx, `UPDATE codex_jobs SET response=? WHERE id=?`, result.Text, id); e != nil {
			return e
		}
		if e = tx.Commit(); e != nil {
			return e
		}
		response = sql.NullString{String: result.Text, Valid: true}
	}
	runes := []rune(response.String)
	for sent < len(runes) {
		end := sent + 3000
		if end > len(runes) {
			end = len(runes)
		}
		if err = s.Send(chat, string(runes[sent:end])); err != nil {
			return err
		}
		sent = end
		if _, err = s.DB.ExecContext(ctx, `UPDATE codex_jobs SET sent_chars=? WHERE id=?`, sent, id); err != nil {
			return err
		}
	}
	_, err = s.DB.ExecContext(ctx, `UPDATE codex_jobs SET done=TRUE WHERE id=?`, id)
	if err == nil {
		_, _ = s.DB.ExecContext(ctx, `DELETE FROM codex_jobs WHERE done=TRUE AND created_at < NOW()-INTERVAL 30 DAY`)
	}
	return err
}
func (s *Service) ask(ctx context.Context, input Request) (Result, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	b, _ := json.Marshal(input)
	req, err := http.NewRequestWithContext(ctx, "POST", s.URL+"/run", bytes.NewReader(b))
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Authorization", "Bearer "+s.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return Result{}, fmt.Errorf("runner status %d", resp.StatusCode)
	}
	var result Result
	err = json.NewDecoder(io.LimitReader(resp.Body, 1024*1024)).Decode(&result)
	return result, err
}
