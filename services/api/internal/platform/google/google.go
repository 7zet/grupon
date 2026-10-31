// Package google exchanges Google OAuth authorization codes for verified identities.
package google

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/grupon/grupon/services/api/internal/domain/identity"
)

const defaultTokenURL = "https://oauth2.googleapis.com/token"

// Config holds the OAuth client settings. All fields are required to enable sign-in.
type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
}

// Client performs the server-side token exchange. The ID token is received directly from Google's
// token endpoint over TLS, so its claims are validated without re-checking the JWKS signature, as
// permitted by the OpenID Connect core specification (section 3.1.3.7).
type Client struct {
	cfg      Config
	tokenURL string
	http     *http.Client
	now      func() time.Time
}

// NewClient returns a client, or nil when the configuration is empty.
func NewClient(cfg Config, now func() time.Time) *Client {
	if cfg.ClientID == "" || cfg.ClientSecret == "" || cfg.RedirectURI == "" {
		return nil
	}
	return &Client{
		cfg:      cfg,
		tokenURL: defaultTokenURL,
		http:     &http.Client{Timeout: 10 * time.Second},
		now:      now,
	}
}

// Exchange trades an authorization code and its PKCE verifier for the verified Google identity.
func (c *Client) Exchange(ctx context.Context, code, codeVerifier, redirectURI string) (identity.GoogleIdentity, error) {
	if redirectURI != c.cfg.RedirectURI {
		return identity.GoogleIdentity{}, identity.ErrGoogleExchangeFailed
	}
	form := url.Values{
		"code":          {code},
		"code_verifier": {codeVerifier},
		"client_id":     {c.cfg.ClientID},
		"client_secret": {c.cfg.ClientSecret},
		"redirect_uri":  {redirectURI},
		"grant_type":    {"authorization_code"},
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return identity.GoogleIdentity{}, fmt.Errorf("build google token request: %w", err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := c.http.Do(request)
	if err != nil {
		return identity.GoogleIdentity{}, fmt.Errorf("google token request: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 64<<10))
	if err != nil {
		return identity.GoogleIdentity{}, fmt.Errorf("read google token response: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return identity.GoogleIdentity{}, identity.ErrGoogleExchangeFailed
	}

	var tokens struct {
		IDToken string `json:"id_token"`
	}
	if err := json.Unmarshal(body, &tokens); err != nil || tokens.IDToken == "" {
		return identity.GoogleIdentity{}, identity.ErrGoogleExchangeFailed
	}
	return c.verifyIDToken(tokens.IDToken)
}

type idClaims struct {
	Email         string `json:"email"`
	EmailVerified any    `json:"email_verified"`
	Name          string `json:"name"`
	jwt.RegisteredClaims
}

func (c *Client) verifyIDToken(raw string) (identity.GoogleIdentity, error) {
	var claims idClaims
	parser := jwt.NewParser(jwt.WithTimeFunc(c.now), jwt.WithExpirationRequired())
	if _, _, err := parser.ParseUnverified(raw, &claims); err != nil {
		return identity.GoogleIdentity{}, identity.ErrGoogleExchangeFailed
	}
	if claims.Issuer != "https://accounts.google.com" && claims.Issuer != "accounts.google.com" {
		return identity.GoogleIdentity{}, errors.Join(identity.ErrGoogleExchangeFailed, errors.New("issuer mismatch"))
	}
	if !audienceContains(claims.RegisteredClaims.Audience, c.cfg.ClientID) {
		return identity.GoogleIdentity{}, errors.Join(identity.ErrGoogleExchangeFailed, errors.New("audience mismatch"))
	}
	if claims.ExpiresAt == nil || !c.now().Before(claims.ExpiresAt.Time) {
		return identity.GoogleIdentity{}, errors.Join(identity.ErrGoogleExchangeFailed, errors.New("id token expired"))
	}
	return identity.GoogleIdentity{
		Subject:       claims.Subject,
		Email:         claims.Email,
		EmailVerified: truthy(claims.EmailVerified),
		Name:          claims.Name,
	}, nil
}

func audienceContains(audience jwt.ClaimStrings, clientID string) bool {
	for _, value := range audience {
		if value == clientID {
			return true
		}
	}
	return false
}

// truthy accepts both boolean and string forms, because Google has returned either over time.
func truthy(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(typed, "true")
	default:
		return false
	}
}
