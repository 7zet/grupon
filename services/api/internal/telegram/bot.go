package telegram

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Config struct{ Token, WebhookSecret, Phone, Telegram, Email string }
type Bot struct {
	cfg    Config
	client *http.Client
	store  LanguageStore
}
type LanguageStore interface {
	SetLanguage(context.Context, int64, string) error
	Language(context.Context, int64) (string, error)
}

func New(cfg Config, stores ...LanguageStore) *Bot {
	if cfg.Token == "" {
		return nil
	}
	var store LanguageStore
	if len(stores) > 0 {
		store = stores[0]
	}
	return &Bot{cfg: cfg, client: &http.Client{Timeout: 10 * time.Second}, store: store}
}

type Update struct {
	UpdateID int      `json:"update_id"`
	Message  *Message `json:"message"`
}
type Message struct {
	Chat Chat   `json:"chat"`
	Text string `json:"text"`
}
type Chat struct {
	ID int64 `json:"id"`
}

func (b *Bot) Webhook(secret string, update Update) error {
	if b.cfg.WebhookSecret == "" {
		return fmt.Errorf("telegram webhook secret is not configured")
	}
	if subtle.ConstantTimeCompare([]byte(secret), []byte(b.cfg.WebhookSecret)) != 1 {
		return fmt.Errorf("invalid telegram webhook secret")
	}
	return b.Handle(context.Background(), update)
}
func (b *Bot) Handle(ctx context.Context, update Update) error {
	if update.Message == nil {
		return nil
	}
	text := strings.ToLower(strings.TrimSpace(update.Message.Text))
	var reply string
	switch text {
	case "/start", "/start uz", "/start ru", "/start en":
		lang := "uz"
		if strings.HasPrefix(text, "/start ") {
			lang = strings.TrimPrefix(text, "/start ")
		}
		if b.store != nil {
			if err := b.store.SetLanguage(ctx, update.Message.Chat.ID, lang); err != nil {
				return err
			}
		}
		reply = "Tilni tanlang: O‘zbekcha / Русский / English\n/help — yordam\n/support — aloqa"
	case "/help":
		reply = "/support buyrug‘i orqali yordam kontaktlarini oling."
	case "/support":
		reply = b.support()
	default:
		return nil
	}
	return b.send(ctx, update.Message.Chat.ID, reply)
}
func (b *Bot) Poll(ctx context.Context) {
	if b == nil {
		return
	}
	offset := 0
	for ctx.Err() == nil {
		updates, err := b.updates(ctx, offset)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
				continue
			}
		}
		for _, update := range updates {
			if update.UpdateID >= offset {
				offset = update.UpdateID + 1
			}
			_ = b.Handle(ctx, update)
		}
	}
}
func (b *Bot) updates(ctx context.Context, offset int) ([]Update, error) {
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?timeout=25&offset=%d", b.cfg.Token, offset), nil)
	if err != nil {
		return nil, err
	}
	res, err := b.client.Do(r)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	var reply struct {
		OK     bool     `json:"ok"`
		Result []Update `json:"result"`
	}
	if err := json.NewDecoder(res.Body).Decode(&reply); err != nil {
		return nil, err
	}
	if !reply.OK {
		return nil, fmt.Errorf("telegram getUpdates failed")
	}
	return reply.Result, nil
}
func (b *Bot) support() string {
	var v []string
	if b.cfg.Phone != "" {
		v = append(v, "Telefon: "+b.cfg.Phone)
	}
	if b.cfg.Telegram != "" {
		v = append(v, "Telegram: "+b.cfg.Telegram)
	}
	if b.cfg.Email != "" {
		v = append(v, "Email: "+b.cfg.Email)
	}
	if len(v) == 0 {
		return "Yordam kontaktlari hali sozlanmagan."
	}
	return strings.Join(v, "\n")
}
func (b *Bot) send(ctx context.Context, chatID int64, text string) error {
	body, _ := json.Marshal(map[string]any{"chat_id": chatID, "text": text})
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.telegram.org/bot"+b.cfg.Token+"/sendMessage", bytes.NewReader(body))
	if err != nil {
		return err
	}
	r.Header.Set("Content-Type", "application/json")
	res, err := b.client.Do(r)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		raw, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
		return fmt.Errorf("telegram send: %s", raw)
	}
	return nil
}
