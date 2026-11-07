package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/grupon/grupon/services/api/internal/domain/identity"
	"github.com/grupon/grupon/services/api/internal/http/gen"
)

const (
	accessCookieName  = "access_token"
	refreshCookieName = "refresh_token"
	// refreshCookiePath covers /api/v1/auth/* (refresh, logout) and /api/v1/me/* (password change,
	// which keeps the current session).
	refreshCookiePath = "/api/v1"
	// maxUserAgentBytes bounds the user agent stored with each refresh token.
	maxUserAgentBytes = 512
)

// AuthOptions configures transport behavior that is not part of the domain.
type AuthOptions struct {
	// SecureCookies sets the Secure attribute. It is enabled outside development.
	SecureCookies bool
	// TrustProxyHeaders reads the client address from X-Real-IP or X-Forwarded-For.
	TrustProxyHeaders bool
	// RefreshTokenTTL sets the refresh cookie lifetime.
	RefreshTokenTTL time.Duration
	// AllowedOrigins lists the web origins that may send cookie-authenticated changes.
	AllowedOrigins []string
	// AllowLocalOrigins also accepts http://localhost and http://127.0.0.1 on any port (development).
	AllowLocalOrigins bool
}

// AuthAPI serves the authentication and account endpoints. Handlers only translate HTTP to the
// identity service and back.
type AuthAPI struct {
	gen.Unimplemented

	identity *identity.Service
	logger   *slog.Logger
	options  AuthOptions
}

var _ gen.ServerInterface = (*AuthAPI)(nil)

// NewAuthAPI binds the identity service to the HTTP contract.
func NewAuthAPI(service *identity.Service, logger *slog.Logger, options AuthOptions) *AuthAPI {
	return &AuthAPI{identity: service, logger: logger, options: options}
}

// RefreshSession rotates the refresh token from the cookie or the bearer header.
func (a *AuthAPI) RefreshSession(w http.ResponseWriter, r *http.Request) {
	session, err := a.identity.Refresh(r.Context(), a.meta(r), refreshTokenFrom(r))
	if err != nil {
		a.clearSessionCookies(w)
		a.fail(w, r, err)
		return
	}
	a.writeSession(w, http.StatusOK, session)
}

// Logout revokes the refresh token of the current session and clears the cookies. It needs only the
// refresh token, so it still works after the access token has expired.
func (a *AuthAPI) Logout(w http.ResponseWriter, r *http.Request) {
	if err := a.identity.Logout(r.Context(), refreshTokenFrom(r)); err != nil {
		a.fail(w, r, err)
		return
	}
	a.clearSessionCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

func (a *AuthAPI) EnrollTwoFactor(w http.ResponseWriter, r *http.Request) {
	var body gen.TwoFactorChallengeRequest
	if _, ok := decodeBody(w, r, &body); !ok {
		return
	}
	enrollment, err := a.identity.BeginTwoFactor(r.Context(), body.ChallengeToken)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, gen.TwoFactorEnrollment{OtpauthUri: enrollment.OTPAuthURI, Secret: enrollment.Secret})
}

func (a *AuthAPI) EnableTwoFactor(w http.ResponseWriter, r *http.Request) {
	var body gen.TwoFactorVerifyRequest
	if _, ok := decodeBody(w, r, &body); !ok {
		return
	}
	session, codes, err := a.identity.EnableTwoFactor(r.Context(), a.meta(r), body.ChallengeToken, body.Code)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	a.setSessionCookies(w, session)
	writeJSON(w, http.StatusOK, gen.TwoFactorEnabled{Session: a.sessionResponse(session), RecoveryCodes: codes})
}

func (a *AuthAPI) VerifyTwoFactor(w http.ResponseWriter, r *http.Request) {
	var body gen.TwoFactorVerifyRequest
	if _, ok := decodeBody(w, r, &body); !ok {
		return
	}
	session, err := a.identity.CompleteTwoFactor(r.Context(), a.meta(r), body.ChallengeToken, body.Code)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	a.writeSession(w, http.StatusOK, session)
}

func (a *AuthAPI) DisableTwoFactor(w http.ResponseWriter, r *http.Request) {
	account, ok := a.authenticate(w, r)
	if !ok {
		return
	}
	var body gen.TwoFactorCode
	if _, ok := decodeBody(w, r, &body); !ok {
		return
	}
	if err := a.identity.DisableTwoFactor(r.Context(), account, body.Code); err != nil {
		a.fail(w, r, err)
		return
	}
	a.clearSessionCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

// GetMe returns the current profile.
func (a *AuthAPI) GetMe(w http.ResponseWriter, r *http.Request) {
	account, ok := a.authenticate(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, toUserResponse(account))
}

// UpdateMe updates profile preferences. An explicit null phone clears the phone number.
func (a *AuthAPI) UpdateMe(w http.ResponseWriter, r *http.Request) {
	account, ok := a.authenticate(w, r)
	if !ok {
		return
	}
	raw, ok := decodeBody(w, r, &gen.UpdateProfileRequest{})
	if !ok {
		return
	}
	var body gen.UpdateProfileRequest
	if err := json.Unmarshal(raw, &body); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", nil, nil)
		return
	}
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(raw, &fields)

	update := identity.ProfileUpdate{Name: body.Name}
	if phone, present := fields["phone"]; present {
		if string(phone) == "null" {
			empty := ""
			update.Phone = &empty
		} else {
			update.Phone = body.Phone
		}
	}
	if body.Locale != nil {
		locale := string(*body.Locale)
		update.Locale = &locale
	}
	if body.Currency != nil {
		currency := string(*body.Currency)
		update.Currency = &currency
	}
	if update.Name == nil && update.Phone == nil && update.Locale == nil && update.Currency == nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", nil, nil)
		return
	}

	updated, err := a.identity.UpdateProfile(r.Context(), account.ID, update)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toUserResponse(updated))
}

// DeleteMe soft-deletes the current account.
func (a *AuthAPI) DeleteMe(w http.ResponseWriter, r *http.Request) {
	account, ok := a.authenticate(w, r)
	if !ok {
		return
	}
	if err := a.identity.DeleteAccount(r.Context(), account.ID); err != nil {
		a.fail(w, r, err)
		return
	}
	a.clearSessionCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

// authenticate resolves the request credentials to an account or writes a 401 problem.
func (a *AuthAPI) authenticate(w http.ResponseWriter, r *http.Request) (identity.Account, bool) {
	account, err := a.identity.Authenticate(r.Context(), accessTokenFrom(r.Context()))
	if err != nil {
		a.fail(w, r, err)
		return identity.Account{}, false
	}
	return account, true
}

// RequireRole returns middleware that admits only accounts holding one of the roles. Roles are
// read from the database on every request, so revoking a role takes effect immediately.
func (a *AuthAPI) RequireRole(roles ...identity.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			account, ok := a.authenticate(w, r)
			if !ok {
				return
			}
			if !account.HasRole(roles...) {
				writeProblem(w, r, http.StatusForbidden, "forbidden", nil, nil)
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), accountKey{}, account)))
		})
	}
}

// AccountFromContext returns the account placed by RequireRole.
func AccountFromContext(ctx context.Context) (identity.Account, bool) {
	account, ok := ctx.Value(accountKey{}).(identity.Account)
	return account, ok
}

func (a *AuthAPI) writeSession(w http.ResponseWriter, status int, session identity.Session) {
	a.setSessionCookies(w, session)
	writeJSON(w, status, a.sessionResponse(session))
}

func (a *AuthAPI) sessionResponse(session identity.Session) gen.AuthSession {
	return gen.AuthSession{
		AccessToken:  session.AccessToken,
		RefreshToken: session.RefreshToken,
		ExpiresIn:    gen.AuthSessionExpiresIn(a.identity.AccessTokenTTL().Seconds()),
		User:         toUserResponse(session.Account),
	}
}

func (a *AuthAPI) setSessionCookies(w http.ResponseWriter, session identity.Session) {
	http.SetCookie(w, a.cookie(accessCookieName, session.AccessToken, "/", int(a.identity.AccessTokenTTL().Seconds())))
	http.SetCookie(w, a.cookie(refreshCookieName, session.RefreshToken, refreshCookiePath, int(a.options.RefreshTokenTTL.Seconds())))
}

func (a *AuthAPI) clearSessionCookies(w http.ResponseWriter) {
	http.SetCookie(w, a.cookie(accessCookieName, "", "/", -1))
	http.SetCookie(w, a.cookie(refreshCookieName, "", refreshCookiePath, -1))
}

func (a *AuthAPI) cookie(name, value, path string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   a.options.SecureCookies,
		SameSite: http.SameSiteLaxMode,
	}
}

func (a *AuthAPI) meta(r *http.Request) identity.RequestMeta {
	userAgent := r.UserAgent()
	if len(userAgent) > maxUserAgentBytes {
		userAgent = userAgent[:maxUserAgentBytes]
	}
	return identity.RequestMeta{
		IP:        clientIP(r, a.options.TrustProxyHeaders),
		UserAgent: userAgent,
	}
}

// fail maps domain errors to stable problem codes. Internal details are logged, not returned.
func (a *AuthAPI) fail(w http.ResponseWriter, r *http.Request, err error) {
	var validation *identity.ValidationError
	switch {
	case errors.As(err, &validation):
		writeProblem(w, r, http.StatusUnprocessableEntity, "validation_failed", nil,
			map[string][]string{validation.Field: {validation.Code}})
	case errors.Is(err, identity.ErrRateLimited):
		w.Header().Set("Retry-After", "900")
		writeProblem(w, r, http.StatusTooManyRequests, "rate_limited", nil, nil)
	case errors.Is(err, identity.ErrEmailTaken):
		writeProblem(w, r, http.StatusConflict, "email_already_registered", nil, nil)
	case errors.Is(err, identity.ErrInvalidCredentials):
		writeProblem(w, r, http.StatusUnauthorized, "invalid_credentials", nil, nil)
	case errors.Is(err, identity.ErrAccountBlocked):
		writeProblem(w, r, http.StatusForbidden, "account_blocked", nil, nil)
	case errors.Is(err, identity.ErrInvalidToken):
		writeProblem(w, r, http.StatusBadRequest, "invalid_token", nil, nil)
	case errors.Is(err, identity.ErrRefreshTokenInvalid):
		writeProblem(w, r, http.StatusUnauthorized, "refresh_token_invalid", nil, nil)
	case errors.Is(err, identity.ErrRefreshTokenReused):
		writeProblem(w, r, http.StatusUnauthorized, "refresh_token_reused", nil, nil)
	case errors.Is(err, identity.ErrUnauthenticated):
		writeProblem(w, r, http.StatusUnauthorized, "unauthenticated", nil, nil)
	case errors.Is(err, identity.ErrTwoFactorInvalid):
		writeProblem(w, r, http.StatusUnauthorized, "two_factor_invalid", nil, nil)
	case errors.Is(err, identity.ErrTwoFactorRequired), errors.Is(err, identity.ErrTwoFactorSetup):
		writeProblem(w, r, http.StatusForbidden, "two_factor_required", nil, nil)
	case errors.Is(err, identity.ErrForbidden):
		writeProblem(w, r, http.StatusForbidden, "forbidden", nil, nil)
	case errors.Is(err, identity.ErrNotFound):
		writeProblem(w, r, http.StatusNotFound, "not_found", nil, nil)
	default:
		a.logger.ErrorContext(r.Context(), "identity request failed",
			slog.String("request_id", middleware.GetReqID(r.Context())),
			slog.String("error", err.Error()),
		)
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", nil, nil)
	}
}

// decodeBody reads a JSON object into dst with unknown fields rejected. It returns the raw body so
// callers that need presence information can reuse it.
func decodeBody(w http.ResponseWriter, r *http.Request, dst any) ([]byte, bool) {
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeProblem(w, r, http.StatusRequestEntityTooLarge, "payload_too_large", nil, nil)
			return nil, false
		}
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", nil, nil)
		return nil, false
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", nil, nil)
		return nil, false
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", nil, nil)
		return nil, false
	}
	return raw, true
}

func toUserResponse(account identity.Account) gen.User {
	roles := make([]gen.Role, 0, len(account.Roles))
	for _, role := range account.Roles {
		roles = append(roles, gen.Role(role))
	}
	return gen.User{
		Id:              uuid.UUID(account.ID),
		Email:           openapi_types.Email(account.Email),
		Name:            account.Name,
		Phone:           account.Phone,
		AvatarUrl:       account.AvatarURL,
		Locale:          gen.Locale(account.Locale),
		Currency:        gen.Currency(account.Currency),
		EmailVerifiedAt: account.EmailVerifiedAt,
		Status:          gen.UserStatus(account.Status),
		Roles:           roles,
	}
}

// refreshTokenFrom returns the refresh token from the cookie, or the bearer value for native clients.
func refreshTokenFrom(r *http.Request) string {
	if value := refreshCookieValue(r); value != "" {
		return value
	}
	return bearerValue(r)
}

func refreshCookieValue(r *http.Request) string {
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func bearerValue(r *http.Request) string {
	header := r.Header.Get("Authorization")
	scheme, value, found := strings.Cut(header, " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return ""
	}
	return strings.TrimSpace(value)
}

// clientIP returns the peer address, or the proxy-provided address when trusted.
func clientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if value := strings.TrimSpace(r.Header.Get("X-Real-IP")); value != "" {
			return value
		}
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			first, _, _ := strings.Cut(forwarded, ",")
			return strings.TrimSpace(first)
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

type accountKey struct{}
type accessTokenKey struct{}

// withCredentials records the access token from the cookie or the bearer header on the request
// context. It does not verify the token; handlers that need an account call authenticate.
func (a *AuthAPI) withCredentials(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerValue(r)
		if token == "" {
			if cookie, err := r.Cookie(accessCookieName); err == nil {
				token = cookie.Value
			}
		}
		if bearerValue(r) == "" && !a.originAllowed(r) {
			writeProblem(w, r, http.StatusForbidden, "origin_not_allowed", nil, nil)
			return
		}
		if token != "" {
			r = r.WithContext(context.WithValue(r.Context(), accessTokenKey{}, token))
		}
		next.ServeHTTP(w, r)
	})
}

func accessTokenFrom(ctx context.Context) string {
	token, _ := ctx.Value(accessTokenKey{}).(string)
	return token
}

// limitRequestBody caps request bodies so that a single request cannot exhaust memory.
func limitRequestBody(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil && r.Body != http.NoBody {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// originAllowed blocks cross-site changes that ride on cookies. Safe methods and requests without
// cookie credentials pass. A browser always sends Origin on cross-origin changes, so a present
// Origin must be a known web origin; a request marked cross-site without Origin is refused.
func (a *AuthAPI) originAllowed(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	if _, err := r.Cookie(accessCookieName); err != nil {
		if _, refreshErr := r.Cookie(refreshCookieName); refreshErr != nil {
			return true
		}
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return r.Header.Get("Sec-Fetch-Site") != "cross-site"
	}
	origin = strings.TrimSuffix(origin, "/")
	for _, allowed := range a.options.AllowedOrigins {
		if origin == strings.TrimSuffix(allowed, "/") {
			return true
		}
	}
	if a.options.AllowLocalOrigins {
		if u, err := url.Parse(origin); err == nil && u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1") {
			return true
		}
	}
	return false
}
