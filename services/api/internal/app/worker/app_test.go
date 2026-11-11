package workerapp

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/dealo/dealo/services/api/internal/platform/config"
)

func TestRunStopsWhenContextIsCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan error, 1)
	go func() {
		logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
		done <- Run(ctx, config.Config{}, logger)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Run() did not stop after context cancellation")
	}
}
