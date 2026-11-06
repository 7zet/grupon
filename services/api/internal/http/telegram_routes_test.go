package httpapi

import (
	"github.com/grupon/grupon/services/api/internal/telegram"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTelegramWebhookRejectsWrongSecret(t *testing.T) {
	bot := telegram.New(telegram.Config{Token: "token", WebhookSecret: "secret"})
	r := NewRouter(RouterOptions{Telegram: NewTelegramAPI(bot, "secret", true)})
	req := httptest.NewRequest(http.MethodPost, "/telegram/webhook", strings.NewReader(`{"update_id":1}`))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "wrong")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", rec.Code)
	}
}
func TestTelegramWebhookIsNotMountedInPollingMode(t *testing.T) {
	bot := telegram.New(telegram.Config{Token: "token"})
	r := NewRouter(RouterOptions{Telegram: NewTelegramAPI(bot, "", false)})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/telegram/webhook", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d", rec.Code)
	}
}
