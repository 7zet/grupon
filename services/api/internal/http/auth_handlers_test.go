package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dealo/dealo/services/api/internal/domain/identity"
	"github.com/dealo/dealo/services/api/internal/domain/identity/identitytest"
)

const (
	httpTestSecret = "http-test-secret-with-at-least-32-bytes"
	httpPassword   = "correct-horse-battery"
)

type authHarness struct {
	router http.Handler
	auth   *AuthAPI
	repo   *identitytest.Repository
}

func newAuthHarness(t *testing.T, limit int, secure bool) authHarness {
	t.Helper()
	repo := identitytest.NewRepository(time.Now)
	mailer := &identitytest.Mailer{}
	tokens, err := identity.NewTokenIssuer(httpTestSecret, 15*time.Minute, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	var limiter identity.RateLimiter
	if limit > 0 {
		limiter = &identitytest.Limiter{Max: limit}
	}
	service := identity.NewService(identity.Dependencies{
		Repository: repo, Tokens: tokens, Mailer: mailer, Limiter: limiter, Now: time.Now,
	}, identity.Config{
		AccessTokenTTL: 15 * time.Minute, RefreshTokenTTL: 30 * 24 * time.Hour,
		VerifyEmailTTL: 24 * time.Hour,
		WebBaseURL:     "https://dealo.test", BrandName: "Dealo",
	})
	auth := NewAuthAPI(service, slog.New(slog.NewTextHandler(io.Discard, nil)), AuthOptions{
		SecureCookies:   secure,
		RefreshTokenTTL: 30 * 24 * time.Hour,
	})
	return authHarness{router: NewRouter(RouterOptions{Auth: auth}), auth: auth, repo: repo}
}

func (h authHarness) do(t *testing.T, method, path string, body any, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(raw)
	}
	request := httptest.NewRequest(method, path, reader)
	request.Header.Set("Content-Type", "application/json")
	request.RemoteAddr = "192.0.2.10:4321"
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	recorder := httptest.NewRecorder()
	h.router.ServeHTTP(recorder, request)
	return recorder
}

func cookieNamed(recorder *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

func authProblemCode(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()
	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/problem+json" {
		t.Fatalf("content type = %q, want application/problem+json", contentType)
	}
	var problem struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &problem); err != nil {
		t.Fatal(err)
	}
	return problem.Code
}

// userID extracts the account id from a session response.
func userID(t *testing.T, recorder *httptest.ResponseRecorder) uuid.UUID {
	t.Helper()
	var session struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &session); err != nil {
		t.Fatal(err)
	}
	id, err := uuid.Parse(session.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func registerBody() map[string]string {
	return map[string]string{
		"email": "shopper@example.test", "password": httpPassword, "name": "Shopper",
		"locale": "en", "currency": "UZS",
	}
}
