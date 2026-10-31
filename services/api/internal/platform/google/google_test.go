package google

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/grupon/grupon/services/api/internal/domain/identity"
)

var now = time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)

func testClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client := NewClient(Config{ClientID: "client-id", ClientSecret: "client-secret", RedirectURI: "https://grupon.test/cb"}, func() time.Time { return now })
	if client == nil {
		t.Fatal("NewClient() = nil with complete config")
	}
	client.tokenURL = server.URL
	return client
}

// idToken builds an unsigned-by-Google token. The client reads claims without signature checks
// because the token is received directly from the token endpoint over TLS.
func idToken(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("local"))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func validClaims() jwt.MapClaims {
	return jwt.MapClaims{
		"iss":            "https://accounts.google.com",
		"aud":            "client-id",
		"sub":            "1234567890",
		"email":          "buyer@example.test",
		"email_verified": true,
		"name":           "Buyer",
		"exp":            now.Add(time.Hour).Unix(),
	}
}

func tokenServer(t *testing.T, claims jwt.MapClaims) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		if r.PostForm.Get("code_verifier") != "verifier-value-that-is-long-enough-43chars" {
			t.Errorf("code_verifier not forwarded")
		}
		if r.PostForm.Get("grant_type") != "authorization_code" || r.PostForm.Get("client_secret") != "client-secret" {
			t.Errorf("unexpected token request form: %v", r.PostForm)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"id_token": idToken(t, claims)})
	}
}

const verifier = "verifier-value-that-is-long-enough-43chars"

func TestExchangeReturnsVerifiedIdentity(t *testing.T) {
	t.Parallel()
	client := testClient(t, tokenServer(t, validClaims()))

	got, err := client.Exchange(context.Background(), "code", verifier, "https://grupon.test/cb")
	if err != nil {
		t.Fatalf("Exchange() error = %v", err)
	}
	want := identity.GoogleIdentity{Subject: "1234567890", Email: "buyer@example.test", EmailVerified: true, Name: "Buyer"}
	if got != want {
		t.Fatalf("Exchange() = %+v, want %+v", got, want)
	}
}

func TestExchangeRejectsBadClaims(t *testing.T) {
	t.Parallel()

	tests := map[string]func(jwt.MapClaims){
		"wrong issuer":   func(c jwt.MapClaims) { c["iss"] = "https://evil.example" },
		"wrong audience": func(c jwt.MapClaims) { c["aud"] = "other-client" },
		"expired":        func(c jwt.MapClaims) { c["exp"] = now.Add(-time.Minute).Unix() },
		"missing expiry": func(c jwt.MapClaims) { delete(c, "exp") },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			claims := validClaims()
			mutate(claims)
			client := testClient(t, tokenServer(t, claims))
			_, err := client.Exchange(context.Background(), "code", verifier, "https://grupon.test/cb")
			if !errors.Is(err, identity.ErrGoogleExchangeFailed) {
				t.Fatalf("Exchange() error = %v, want ErrGoogleExchangeFailed", err)
			}
		})
	}
}

func TestExchangeRejectsForeignRedirectURI(t *testing.T) {
	t.Parallel()
	client := testClient(t, tokenServer(t, validClaims()))

	_, err := client.Exchange(context.Background(), "code", verifier, "https://evil.example/cb")
	if !errors.Is(err, identity.ErrGoogleExchangeFailed) {
		t.Fatalf("error = %v, want ErrGoogleExchangeFailed", err)
	}
}

func TestExchangeFailsOnGoogleError(t *testing.T) {
	t.Parallel()
	client := testClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
	})

	_, err := client.Exchange(context.Background(), "code", verifier, "https://grupon.test/cb")
	if !errors.Is(err, identity.ErrGoogleExchangeFailed) {
		t.Fatalf("error = %v, want ErrGoogleExchangeFailed", err)
	}
}

func TestNewClientRequiresCompleteConfig(t *testing.T) {
	t.Parallel()
	if NewClient(Config{ClientID: "id"}, time.Now) != nil {
		t.Fatal("NewClient() with partial config = non-nil")
	}
}
