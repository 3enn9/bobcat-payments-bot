package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMaxWebhookRequiresSecret(t *testing.T) {
	t.Setenv("MAX_WEBHOOK_SECRET", "private-secret")
	h := &MaxHandler{}
	for _, secret := range []string{"", "incorrect"} {
		req := httptest.NewRequest("POST", "/max/webhook", strings.NewReader(`{}`))
		req.Header.Set("X-Max-Bot-Api-Secret", secret)
		w := httptest.NewRecorder()
		h.MaxUpdates(w, req)
		if w.Code != 401 {
			t.Fatalf("unauthorized webhook accepted: %d", w.Code)
		}
	}
}
