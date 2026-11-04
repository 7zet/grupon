package httpapi

import (
	"context"
	"crypto/subtle"
	"net/http"

	"github.com/go-chi/chi/v5"
)

const internalTokenHeader = "X-Internal-Token"

// InternalAPI exposes operational hooks that are not part of the public product contract.
// It is mounted only when a token is configured, and every request must present that token.
type InternalAPI struct {
	token      string
	invalidate func(context.Context) error
}

// NewInternalAPI builds the operational API. An empty token disables the endpoints entirely.
func NewInternalAPI(token string, invalidate func(context.Context) error) *InternalAPI {
	return &InternalAPI{token: token, invalidate: invalidate}
}

func (a *InternalAPI) mount(router chi.Router) {
	if a.token == "" {
		return
	}
	router.Post("/internal/cache/invalidate", a.invalidateCache)
}

func (a *InternalAPI) invalidateCache(w http.ResponseWriter, r *http.Request) {
	presented := r.Header.Get(internalTokenHeader)
	if subtle.ConstantTimeCompare([]byte(presented), []byte(a.token)) != 1 {
		writeProblem(w, r, http.StatusUnauthorized, "unauthorized", nil, nil)
		return
	}
	if err := a.invalidate(r.Context()); err != nil {
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", nil, nil)
		return
	}
	writeStatus(w, http.StatusAccepted, "invalidated")
}
