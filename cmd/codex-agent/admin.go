package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	mysql "github.com/go-sql-driver/mysql"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

func adminEnabled() bool { return os.Getenv("CODEX_PROJECT_ADMIN") == "true" }
func executeSQL(query string) (string, error) {
	if !adminEnabled() {
		return "", fmt.Errorf("project administration disabled")
	}
	if strings.TrimSpace(query) == "" {
		return "", fmt.Errorf("empty SQL")
	}
	cfg := mysql.NewConfig()
	cfg.User = os.Getenv("AGENT_DB_USER")
	cfg.Passwd = os.Getenv("AGENT_DB_PASS")
	cfg.Net = "tcp"
	cfg.Addr = os.Getenv("AGENT_DB_ADDR")
	cfg.DBName = os.Getenv("AGENT_DB_NAME")
	cfg.Timeout = 3 * time.Second
	cfg.ReadTimeout = 35 * time.Second
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return "", err
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	result, err := db.ExecContext(ctx, query)
	if err != nil {
		return "", err
	}
	rows, _ := result.RowsAffected()
	id, _ := result.LastInsertId()
	return fmt.Sprintf("SQL executed in PaymentsBot database. affected_rows=%d last_insert_id=%d", rows, id), nil
}
func controlProject(args map[string]any) (string, error) {
	if !adminEnabled() {
		return "", fmt.Errorf("project administration disabled")
	}
	data, err := json.Marshal(args)
	if err != nil {
		return "", err
	}
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", "/control/control.sock")
	}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 20 * time.Second}
	response, err := client.Post("http://control/", "application/json", bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 64000))
	if err != nil {
		return "", err
	}
	if response.StatusCode != 200 {
		return "", fmt.Errorf("project control: %s", body)
	}
	return string(body), nil
}
