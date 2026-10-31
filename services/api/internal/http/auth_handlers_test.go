package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/grupon/grupon/services/api/internal/domain/identity"
	"github.com/grupon/grupon/services/api/internal/domain/identity/identitytest"
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
		VerifyEmailTTL: 24 * time.Hour, ResetPasswordTTL: time.Hour,
		WebBaseURL: "https://grupon.test", BrandName: "Grupon",
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

func TestRegisterSetsSecureCookiesAndReturnsSession(t *testing.T) {
	t.Parallel()
	h := newAuthHarness(t, 0, true)

	recorder := h.do(t, http.MethodPost, "/api/v1/auth/register", registerBody())
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	access := cookieNamed(recorder, accessCookieName)
	refresh := cookieNamed(recorder, refreshCookieName)
	if access == nil || refresh == nil {
		t.Fatal("register did not set both session cookies")
	}
	for _, cookie := range []*http.Cookie{access, refresh} {
		if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode {
			t.Errorf("cookie %s flags: HttpOnly=%v Secure=%v SameSite=%v", cookie.Name, cookie.HttpOnly, cookie.Secure, cookie.SameSite)
		}
	}
	if access.Path != "/" || refresh.Path != "/api/v1" {
		t.Errorf("cookie paths = %q / %q", access.Path, refresh.Path)
	}

	var session struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		User        struct {
			Email string   `json:"email"`
			Roles []string `json:"roles"`
		} `json:"user"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &session); err != nil {
		t.Fatal(err)
	}
	if session.ExpiresIn != 900 || session.AccessToken == "" || session.User.Email != "shopper@example.test" {
		t.Errorf("session = %+v", session)
	}
	if len(session.User.Roles) != 1 || session.User.Roles[0] != "user" {
		t.Errorf("roles = %v, want [user]", session.User.Roles)
	}
}

func TestCookiesAreNotSecureInDevelopment(t *testing.T) {
	t.Parallel()
	h := newAuthHarness(t, 0, false)

	recorder := h.do(t, http.MethodPost, "/api/v1/auth/register", registerBody())
	if cookie := cookieNamed(recorder, accessCookieName); cookie == nil || cookie.Secure {
		t.Fatalf("access cookie = %+v, want non-secure in development", cookie)
	}
}

func TestLoginErrorIsUniform(t *testing.T) {
	t.Parallel()
	h := newAuthHarness(t, 0, false)
	h.do(t, http.MethodPost, "/api/v1/auth/register", registerBody())

	wrong := h.do(t, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": "shopper@example.test", "password": "not-the-password-1"})
	unknown := h.do(t, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": "ghost@example.test", "password": "not-the-password-1"})

	if wrong.Code != http.StatusUnauthorized || unknown.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d / %d, want 401", wrong.Code, unknown.Code)
	}
	if authProblemCode(t, wrong) != "invalid_credentials" || authProblemCode(t, unknown) != "invalid_credentials" {
		t.Fatal("login failures do not share the invalid_credentials code")
	}
	if wrong.Body.String() != unknown.Body.String() {
		t.Fatalf("problem bodies differ:\n%s\n%s", wrong.Body.String(), unknown.Body.String())
	}
}

func TestLoginIsRateLimitedWith429(t *testing.T) {
	t.Parallel()
	h := newAuthHarness(t, 3, false)
	h.do(t, http.MethodPost, "/api/v1/auth/register", registerBody())

	body := map[string]string{"email": "shopper@example.test", "password": "wrong-password-value"}
	for i := 0; i < 3; i++ {
		h.do(t, http.MethodPost, "/api/v1/auth/login", body)
	}
	recorder := h.do(t, http.MethodPost, "/api/v1/auth/login", body)
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", recorder.Code)
	}
	if code := authProblemCode(t, recorder); code != "rate_limited" {
		t.Fatalf("code = %q, want rate_limited", code)
	}
	if recorder.Header().Get("Retry-After") == "" {
		t.Error("429 without Retry-After")
	}
}

func TestRefreshRotatesAndReuseReturns401(t *testing.T) {
	t.Parallel()
	h := newAuthHarness(t, 0, false)
	registered := h.do(t, http.MethodPost, "/api/v1/auth/register", registerBody())
	oldRefresh := cookieNamed(registered, refreshCookieName)

	rotated := h.do(t, http.MethodPost, "/api/v1/auth/refresh", nil, oldRefresh)
	if rotated.Code != http.StatusOK {
		t.Fatalf("refresh status = %d, body = %s", rotated.Code, rotated.Body.String())
	}
	if newRefresh := cookieNamed(rotated, refreshCookieName); newRefresh == nil || newRefresh.Value == oldRefresh.Value {
		t.Fatal("refresh did not rotate the cookie")
	}

	reused := h.do(t, http.MethodPost, "/api/v1/auth/refresh", nil, oldRefresh)
	if reused.Code != http.StatusUnauthorized || authProblemCode(t, reused) != "refresh_token_reused" {
		t.Fatalf("reuse = %d %s, want 401 refresh_token_reused", reused.Code, reused.Body.String())
	}
	if h.repo.ActiveRefreshCount(userID(t, registered)) != 0 {
		t.Error("reuse did not revoke the family")
	}
}

func TestMeRequiresAuthenticationAndLogoutClearsCookies(t *testing.T) {
	t.Parallel()
	h := newAuthHarness(t, 0, false)
	registered := h.do(t, http.MethodPost, "/api/v1/auth/register", registerBody())
	access := cookieNamed(registered, accessCookieName)
	refresh := cookieNamed(registered, refreshCookieName)

	if anonymous := h.do(t, http.MethodGet, "/api/v1/me", nil); anonymous.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous /me status = %d, want 401", anonymous.Code)
	}
	if me := h.do(t, http.MethodGet, "/api/v1/me", nil, access); me.Code != http.StatusOK {
		t.Fatalf("/me status = %d, body = %s", me.Code, me.Body.String())
	}

	logout := h.do(t, http.MethodPost, "/api/v1/auth/logout", nil, access, refresh)
	if logout.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d", logout.Code)
	}
	if cleared := cookieNamed(logout, refreshCookieName); cleared == nil || cleared.MaxAge >= 0 {
		t.Error("logout did not clear the refresh cookie")
	}
	if after := h.do(t, http.MethodPost, "/api/v1/auth/refresh", nil, refresh); after.Code != http.StatusUnauthorized {
		t.Fatalf("refresh after logout = %d, want 401", after.Code)
	}
}

func TestBearerTokenIsAcceptedForNativeClients(t *testing.T) {
	t.Parallel()
	h := newAuthHarness(t, 0, false)
	registered := h.do(t, http.MethodPost, "/api/v1/auth/register", registerBody())
	var session struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(registered.Body.Bytes(), &session); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	request.Header.Set("Authorization", "Bearer "+session.AccessToken)
	recorder := httptest.NewRecorder()
	h.router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("bearer /me status = %d", recorder.Code)
	}
}

func TestRequireRoleDeniesWithoutRoleAndAdmitsWithRole(t *testing.T) {
	t.Parallel()
	h := newAuthHarness(t, 0, false)

	guarded := chi.NewRouter()
	guarded.Use(h.auth.withCredentials)
	guarded.With(h.auth.RequireRole(identity.RoleAdmin)).Get("/internal/guarded", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	registered := h.do(t, http.MethodPost, "/api/v1/auth/register", registerBody())
	access := cookieNamed(registered, accessCookieName)

	request := httptest.NewRequest(http.MethodGet, "/internal/guarded", nil)
	request.AddCookie(access)
	denied := httptest.NewRecorder()
	guarded.ServeHTTP(denied, request)
	if denied.Code != http.StatusForbidden || authProblemCode(t, denied) != "forbidden" {
		t.Fatalf("user without admin role: status = %d", denied.Code)
	}

	h.repo.GrantRole(userID(t, registered), identity.RoleAdmin)
	request = httptest.NewRequest(http.MethodGet, "/internal/guarded", nil)
	request.AddCookie(access)
	allowed := httptest.NewRecorder()
	guarded.ServeHTTP(allowed, request)
	if allowed.Code != http.StatusNoContent {
		t.Fatalf("admin: status = %d, want 204", allowed.Code)
	}
}

func TestOversizedBodyIsRejected(t *testing.T) {
	t.Parallel()
	h := newAuthHarness(t, 0, false)
	payload := `{"email":"a@example.test","password":"` + strings.Repeat("x", 70<<10) + `"}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(payload))
	recorder := httptest.NewRecorder()
	h.router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", recorder.Code)
	}
}

func TestUnknownFieldsAreRejected(t *testing.T) {
	t.Parallel()
	h := newAuthHarness(t, 0, false)
	body := map[string]string{"email": "a@example.test", "password": "x", "role": "admin"}
	recorder := h.do(t, http.MethodPost, "/api/v1/auth/login", body)
	if recorder.Code != http.StatusBadRequest || authProblemCode(t, recorder) != "invalid_request" {
		t.Fatalf("status = %d, want 400 invalid_request", recorder.Code)
	}
}

func TestForgotPasswordAlwaysAccepted(t *testing.T) {
	t.Parallel()
	h := newAuthHarness(t, 0, false)

	// An unknown but well-formed address is accepted exactly like a registered one.
	ghost := h.do(t, http.MethodPost, "/api/v1/auth/forgot-password", map[string]string{"email": "ghost@example.test"})
	if ghost.Code != http.StatusAccepted {
		t.Errorf("unknown email status = %d, want 202", ghost.Code)
	}
	// A malformed address fails the contract's email format before reaching the service.
	malformed := h.do(t, http.MethodPost, "/api/v1/auth/forgot-password", map[string]string{"email": "malformed"})
	if malformed.Code != http.StatusBadRequest {
		t.Errorf("malformed email status = %d, want 400", malformed.Code)
	}
}

func TestPasswordChangeRequiresCurrentPassword(t *testing.T) {
	t.Parallel()
	h := newAuthHarness(t, 0, false)
	registered := h.do(t, http.MethodPost, "/api/v1/auth/register", registerBody())
	access := cookieNamed(registered, accessCookieName)

	recorder := h.do(t, http.MethodPut, "/api/v1/me/password", map[string]string{
		"current_password": "wrong-current-password", "new_password": "brand-new-password-1",
	}, access)
	if recorder.Code != http.StatusUnprocessableEntity || authProblemCode(t, recorder) != "current_password_incorrect" {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
}
