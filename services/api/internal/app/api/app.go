// Package apiapp owns the API process lifecycle.
package apiapp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	httpapi "github.com/grupon/grupon/services/api/internal/http"
	"github.com/grupon/grupon/services/api/internal/platform/config"
)

// Run starts the HTTP server and blocks until it stops or the context is canceled.
func Run(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", cfg.HTTP.Address)
	if err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return fmt.Errorf("listen on %s: %w", cfg.HTTP.Address, err)
	}

	router := httpapi.NewRouter(httpapi.RouterOptions{Logger: logger})
	server := newServer(cfg.HTTP, router, logger)

	logger.InfoContext(ctx, "api server started", slog.String("address", listener.Addr().String()))
	return serve(ctx, server, listener, cfg.ShutdownTimeout, logger)
}

func newServer(cfg config.HTTPConfig, handler http.Handler, logger *slog.Logger) *http.Server {
	return &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}
}

func serve(
	ctx context.Context,
	server *http.Server,
	listener net.Listener,
	shutdownTimeout time.Duration,
	logger *slog.Logger,
) error {
	serveErrors := make(chan error, 1)
	go func() {
		serveErrors <- server.Serve(listener)
	}()

	select {
	case err := <-serveErrors:
		return unexpectedServeError(err)
	case <-ctx.Done():
		logger.Info("api server stopping")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		shutdownErr := server.Shutdown(shutdownCtx)
		if shutdownErr != nil {
			closeErr := server.Close()
			serveErr := unexpectedServeError(<-serveErrors)
			return errors.Join(
				fmt.Errorf("shut down api server: %w", shutdownErr),
				closeErr,
				serveErr,
			)
		}

		if err := unexpectedServeError(<-serveErrors); err != nil {
			return err
		}

		logger.Info("api server stopped")
		return nil
	}
}

func unexpectedServeError(err error) error {
	if err == nil || errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return fmt.Errorf("serve http: %w", err)
}
