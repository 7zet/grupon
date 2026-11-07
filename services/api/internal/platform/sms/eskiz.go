package sms

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const DefaultEskizBaseURL = "https://notify.eskiz.uz/api"

type EskizSender struct {
	base, email, password, sender string
	client                        *http.Client
	mu                            sync.Mutex
	token                         string
}

func NewEskizSender(base, email, password, sender string) (*EskizSender, error) {
	if base == "" {
		base = DefaultEskizBaseURL
	}
	if email == "" || password == "" || sender == "" {
		return nil, fmt.Errorf("Eskiz credentials and sender are required")
	}
	return &EskizSender{strings.TrimSuffix(base, "/"), email, password, sender, &http.Client{Timeout: 10 * time.Second}, sync.Mutex{}, ""}, nil
}
func (s *EskizSender) SendCode(ctx context.Context, phone, code string) error {
	err := s.send(ctx, phone, code)
	if err != nil && strings.Contains(err.Error(), "401") {
		s.mu.Lock()
		s.token = ""
		s.mu.Unlock()
		return s.send(ctx, phone, code)
	}
	return err
}
func (s *EskizSender) send(ctx context.Context, phone, code string) error {
	token, err := s.auth(ctx)
	if err != nil {
		return err
	}
	v := url.Values{"mobile_phone": {strings.TrimPrefix(phone, "+")}, "message": {"Grupon: tasdiqlash kodi " + code}, "from": {s.sender}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.base+"/message/sms/send", strings.NewReader(v.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		return fmt.Errorf("eskiz send: %d", res.StatusCode)
	}
	return nil
}
func (s *EskizSender) auth(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.token != "" {
		return s.token, nil
	}
	v := url.Values{"email": {s.email}, "password": {s.password}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.base+"/auth/login", strings.NewReader(v.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		return "", fmt.Errorf("eskiz auth: %d", res.StatusCode)
	}
	var out struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.Data.Token == "" {
		return "", fmt.Errorf("eskiz auth: empty token")
	}
	s.token = out.Data.Token
	return s.token, nil
}
