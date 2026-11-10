package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	apiapp "github.com/dealo/dealo/services/api/internal/app/api"
	"github.com/dealo/dealo/services/api/internal/platform/config"
	"github.com/dealo/dealo/services/api/internal/platform/logging"
)

func main() {
	os.Exit(run())
}

func run() int {
	bootstrapLogger := logging.New(os.Stdout, slog.LevelInfo)
	cfg, err := config.Load()
	if err != nil {
		bootstrapLogger.Error("load configuration", slog.String("error", err.Error()))
		return 1
	}

	logger := logging.New(os.Stdout, cfg.LogLevel).With(
		slog.String("service", "api"),
		slog.String("environment", cfg.Environment),
	)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := apiapp.Run(ctx, cfg, logger); err != nil {
		logger.Error("api process stopped with an error", slog.String("error", err.Error()))
		return 1
	}

	return 0
}
