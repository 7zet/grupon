// Package httpapi contains the HTTP transport and its operational endpoints.
package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const requestIDHeader = "X-Request-ID"

// ReadinessCheck reports whether the process can serve traffic.
type ReadinessCheck func(context.Context) error

// RouterOptions supplies infrastructure dependencies to the HTTP transport.
type RouterOptions struct {
	Logger         *slog.Logger
	ReadinessCheck ReadinessCheck
}

// NewRouter creates the application router. Business routes are added in later phases.
func NewRouter(options RouterOptions) chi.Router {
	logger := options.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	readinessCheck := options.ReadinessCheck
	if readinessCheck == nil {
		readinessCheck = func(context.Context) error { return nil }
	}

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(addRequestIDHeader)
	router.Use(logRequests(logger))
	router.Use(recoverPanics(logger))

	router.Get("/healthz", statusHandler(http.StatusOK, "ok"))
	router.Get("/readyz", readinessHandler(readinessCheck))

	return router
}

func readinessHandler(check ReadinessCheck) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		if err := check(request.Context()); err != nil {
			writeStatus(response, http.StatusServiceUnavailable, "not_ready")
			return
		}

		writeStatus(response, http.StatusOK, "ok")
	}
}

func statusHandler(code int, status string) http.HandlerFunc {
	return func(response http.ResponseWriter, _ *http.Request) {
		writeStatus(response, code, status)
	}
}

func writeStatus(response http.ResponseWriter, code int, status string) {
	response.Header().Set("Cache-Control", "no-store")
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(code)
	_ = json.NewEncoder(response).Encode(struct {
		Status string `json:"status"`
	}{Status: status})
}

func addRequestIDHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if requestID := middleware.GetReqID(request.Context()); requestID != "" {
			response.Header().Set(requestIDHeader, requestID)
		}

		next.ServeHTTP(response, request)
	})
}

func logRequests(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			startedAt := time.Now()
			wrapped := middleware.NewWrapResponseWriter(response, request.ProtoMajor)

			next.ServeHTTP(wrapped, request)

			status := wrapped.Status()
			if status == 0 {
				status = http.StatusOK
			}
			logger.InfoContext(request.Context(), "http request completed",
				slog.String("request_id", middleware.GetReqID(request.Context())),
				slog.String("method", request.Method),
				slog.String("path", request.URL.Path),
				slog.Int("status", status),
				slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
			)
		})
	}
}

func recoverPanics(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.ErrorContext(request.Context(), "http panic recovered",
						slog.String("request_id", middleware.GetReqID(request.Context())),
					)
					writeStatus(response, http.StatusInternalServerError, "internal_error")
				}
			}()

			next.ServeHTTP(response, request)
		})
	}
}
