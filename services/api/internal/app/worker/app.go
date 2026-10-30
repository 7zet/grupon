// Package workerapp owns the background worker process lifecycle.
package workerapp

import (
	"context"
	"log/slog"
)

// Run keeps the worker process alive until shutdown. Job registration starts in a later phase.
func Run(ctx context.Context, logger *slog.Logger) error {
	logger.InfoContext(ctx, "worker started")
	<-ctx.Done()
	logger.Info("worker stopped")

	return nil
}
