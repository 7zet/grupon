package httpapi

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/grupon/grupon/services/api/internal/telegram"
	"net/http"
)

type TelegramAPI struct {
	bot         *telegram.Bot
	secret      string
	webhookMode bool
}

func NewTelegramAPI(bot *telegram.Bot, secret string, webhookMode bool) *TelegramAPI {
	return &TelegramAPI{bot: bot, secret: secret, webhookMode: webhookMode}
}
func (a *TelegramAPI) mount(r chi.Router) {
	if a.bot != nil && a.webhookMode {
		r.Post("/telegram/webhook", a.webhook)
	}
}
func (a *TelegramAPI) webhook(w http.ResponseWriter, r *http.Request) {
	var u telegram.Update
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_telegram_update", nil, nil)
		return
	}
	if err := a.bot.Webhook(r.Header.Get("X-Telegram-Bot-Api-Secret-Token"), u); err != nil {
		writeProblem(w, r, http.StatusUnauthorized, "invalid_telegram_webhook", nil, nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
