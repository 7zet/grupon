// Package workerapp owns the background worker process lifecycle.
package workerapp

import (
	"context"
	"fmt"
	"github.com/dealo/dealo/services/api/internal/domain/finance"
	"github.com/dealo/dealo/services/api/internal/domain/outbox"
	"log/slog"
	"time"

	"github.com/dealo/dealo/services/api/internal/app/commerceapp"
	"github.com/dealo/dealo/services/api/internal/platform/config"
	"github.com/dealo/dealo/services/api/internal/platform/database"
	"github.com/dealo/dealo/services/api/internal/store"
)

const (
	// sweepInterval is how often expired reservations and vouchers are processed.
	sweepInterval = 30 * time.Second
	// sweepBatch bounds the rows one sweep step locks at a time.
	sweepBatch = 100
)

// Run executes the periodic sweeps until ctx is canceled. Without a database URL it only waits, which
// keeps the process healthy for orchestration checks.
func Run(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	logger.InfoContext(ctx, "worker started")
	if cfg.DatabaseURL == "" {
		<-ctx.Done()
		logger.Info("worker stopped")
		return nil
	}

	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer pool.Close()

	commerce, err := commerceapp.New(cfg, pool, store.NewCatalogRepository(pool), logger)
	if err != nil {
		return err
	}

	deals := store.NewDealJobs(pool)
	financeJobs := finance.NewService(store.NewFinanceRepository(pool), time.Now)
	outboxRelay := outbox.NewRelay(store.NewOutboxRepository(pool), logPublisher(logger))
	ticker := time.NewTicker(sweepInterval)
	defer ticker.Stop()
	for {
		runSweeps(ctx, commerce, deals, financeJobs, logger)
		if published, err := outboxRelay.Run(ctx, sweepBatch); err != nil {
			logger.ErrorContext(ctx, "outbox relay failed", slog.String("error", err.Error()))
		} else if published > 0 {
			logger.InfoContext(ctx, "outbox events published", slog.Int("events", published))
		}
		select {
		case <-ctx.Done():
			logger.Info("worker stopped")
			return nil
		case <-ticker.C:
		}
	}
}

// logPublisher is the transport until a broker exists: events are written to the log so they can be traced.
func logPublisher(logger *slog.Logger) outbox.Publisher {
	return func(ctx context.Context, topic string, payload []byte) error {
		logger.InfoContext(ctx, "outbox event", slog.String("topic", topic), slog.String("payload", string(payload)))
		return nil
	}
}

// runSweeps logs failures and continues: a broken sweep must not stop the others.
func runSweeps(ctx context.Context, sweeps interface {
	ReleaseExpiredReservations(context.Context, int) (int, error)
	ExpireVouchers(context.Context, int) (int, error)
}, deals interface {
	RunDealSweeps(context.Context) (store.DealSweepResult, error)
}, financeJobs interface {
	RunBreakage(context.Context, int) (int, error)
	Reconcile(context.Context) (bool, error)
}, logger *slog.Logger) {
	if ctx.Err() != nil {
		return
	}
	cancelled, err := sweeps.ReleaseExpiredReservations(ctx, sweepBatch)
	if err != nil {
		logger.ErrorContext(ctx, "release expired reservations failed", slog.String("error", err.Error()))
	} else if cancelled > 0 {
		logger.InfoContext(ctx, "expired reservations released", slog.Int("orders_cancelled", cancelled))
	}
	expired, err := sweeps.ExpireVouchers(ctx, sweepBatch)
	if err != nil {
		logger.ErrorContext(ctx, "expire vouchers failed", slog.String("error", err.Error()))
	} else if expired > 0 {
		logger.InfoContext(ctx, "vouchers expired", slog.Int("count", expired))
	}
	if _, err := deals.RunDealSweeps(ctx); err != nil {
		logger.ErrorContext(ctx, "deal sweeps failed", slog.String("error", err.Error()))
	}
	if moved, err := financeJobs.RunBreakage(ctx, sweepBatch); err != nil {
		logger.ErrorContext(ctx, "breakage failed", slog.String("error", err.Error()))
	} else if moved > 0 {
		logger.InfoContext(ctx, "breakage recognised", slog.Int("vouchers", moved))
	}
	if ok, err := financeJobs.Reconcile(ctx); err != nil {
		logger.ErrorContext(ctx, "reconciliation failed", slog.String("error", err.Error()))
	} else if !ok {
		logger.ErrorContext(ctx, "reconciliation mismatch recorded for finance review")
	}
}
