package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOriginAllowed(t *testing.T) {
	t.Parallel()

	api := &AuthAPI{options: AuthOptions{AllowedOrigins: []string{"https://shop.example"}}}
	local := &AuthAPI{options: AuthOptions{AllowLocalOrigins: true}}
	tests := []struct {
		name    string
		api     *AuthAPI
		method  string
		cookie  bool
		origin  string
		fetch   string
		allowed bool
	}{
		{"safe method", api, http.MethodGet, true, "https://evil.example", "", true},
		{"no cookie", api, http.MethodPost, false, "https://evil.example", "", true},
		{"known origin", api, http.MethodPost, true, "https://shop.example", "", true},
		{"foreign origin", api, http.MethodPost, true, "https://evil.example", "", false},
		{"cross-site without origin", api, http.MethodPost, true, "", "cross-site", false},
		{"same-origin without origin", api, http.MethodPost, true, "", "same-origin", true},
		{"local origin in development", local, http.MethodPost, true, "http://127.0.0.1:3000", "", true},
		{"local origin outside development", api, http.MethodPost, true, "http://localhost:3000", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := httptest.NewRequest(tt.method, "/api/v1/orders", nil)
			if tt.cookie {
				r.AddCookie(&http.Cookie{Name: accessCookieName, Value: "token"})
			}
			if tt.origin != "" {
				r.Header.Set("Origin", tt.origin)
			}
			if tt.fetch != "" {
				r.Header.Set("Sec-Fetch-Site", tt.fetch)
			}
			if got := tt.api.originAllowed(r); got != tt.allowed {
				t.Fatalf("originAllowed() = %v, want %v", got, tt.allowed)
			}
		})
	}
}
