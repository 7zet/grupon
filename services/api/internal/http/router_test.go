package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOperationalEndpoints(t *testing.T) {
	t.Parallel()

	router := NewRouter(RouterOptions{Logger: discardLogger()})

	for _, path := range []string{"/healthz", "/readyz"} {
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequest(http.MethodGet, path, nil)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != http.StatusOK {
				t.Errorf("status code = %d, want %d", response.Code, http.StatusOK)
			}
			if got := response.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", got)
			}
			if got := response.Header().Get(requestIDHeader); got == "" {
				t.Error("request ID response header is empty")
			}

			var body struct {
				Status string `json:"status"`
			}
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if body.Status != "ok" {
				t.Errorf("status = %q, want ok", body.Status)
			}
		})
	}
}

func TestReadinessFailureDoesNotExposeCause(t *testing.T) {
	t.Parallel()

	const privateCause = "database password was rejected"
	router := NewRouter(RouterOptions{
		Logger: discardLogger(),
		ReadinessCheck: func(context.Context) error {
			return errors.New(privateCause)
		},
	})
	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Errorf("status code = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
	if body := response.Body.String(); body != "{\"status\":\"not_ready\"}\n" {
		t.Errorf("body = %q, want sanitized not-ready response", body)
	}
}

func TestRouterRecoversPanics(t *testing.T) {
	t.Parallel()

	router := NewRouter(RouterOptions{Logger: discardLogger()})
	router.Get("/panic", func(http.ResponseWriter, *http.Request) {
		panic("sensitive detail")
	})
	request := httptest.NewRequest(http.MethodGet, "/panic", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Errorf("status code = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if body := response.Body.String(); body != "{\"status\":\"internal_error\"}\n" {
		t.Errorf("body = %q, want sanitized internal-error response", body)
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}
