package codexagent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRunnerRequestPreservesSessionAndAuthentication(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer service-secret" {
			t.Error("missing auth")
		}
		var input Request
		if json.NewDecoder(r.Body).Decode(&input) != nil || input.Session != "thread-123" || input.Prompt != "Проверь счета" {
			t.Error("request lost context")
		}
		json.NewEncoder(w).Encode(Result{Text: "Ответ", Session: "thread-123"})
	}))
	defer server.Close()
	s := Service{URL: server.URL, Token: "service-secret"}
	result, err := s.ask(context.Background(), Request{"Проверь счета", "thread-123"})
	if err != nil || result.Text != "Ответ" || result.Session != "thread-123" {
		t.Fatalf("result=%+v error=%v", result, err)
	}
}
func TestRunnerFailureIsNotAValidAnswer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "auth unavailable", 503) }))
	defer server.Close()
	s := Service{URL: server.URL}
	if _, err := s.ask(context.Background(), Request{Prompt: "test"}); err == nil {
		t.Fatal("runner failure silently accepted")
	}
}
