package sms

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestEskizSenderSendsAndRefreshesToken(t *testing.T) {
	var logins, sends atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login":
			logins.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"token":"token"}}`))
		case "/message/sms/send":
			if sends.Add(1) == 1 {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if r.Header.Get("Authorization") != "Bearer token" {
				t.Errorf("authorization = %q", r.Header.Get("Authorization"))
			}
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("path = %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	sender, err := NewEskizSender(server.URL, "mail@example.test", "password", "GRUPON")
	if err != nil {
		t.Fatal(err)
	}
	if err := sender.SendCode(context.Background(), "+998901234567", "123456"); err != nil {
		t.Fatal(err)
	}
	if sends.Load() != 2 || logins.Load() != 2 {
		t.Fatalf("sends=%d logins=%d", sends.Load(), logins.Load())
	}
}

func TestEskizSenderReturnsProviderError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/login" {
			_, _ = w.Write([]byte(`{"data":{"token":"token"}}`))
			return
		}
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()
	sender, err := NewEskizSender(server.URL, "mail@example.test", "password", "GRUPON")
	if err != nil {
		t.Fatal(err)
	}
	if err := sender.SendCode(context.Background(), "+998901234567", "123456"); err == nil {
		t.Fatal("SendCode error = nil")
	}
}
