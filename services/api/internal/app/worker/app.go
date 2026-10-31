// Package workerapp owns the background worker process lifecycle.
package workerapp

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/grupon/grupon/services/api/internal/app/commerceapp"
	"github.com/grupon/grupon/services/api/internal/platform/config"
	"github.com/grupon/grupon/services/api/internal/platform/database"
	"github.com/grupon/grupon/services/api/internal/store"
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
	ticker := time.NewTicker(sweepInterval)
	defer ticker.Stop()
	for {
		runSweeps(ctx, commerce, deals, logger)
		select {
		case <-ctx.Done():
			logger.Info("worker stopped")
			return nil
		case <-ticker.C:
		}
	}
}

// runSweeps logs failures and continues: a broken sweep must not stop the others.
func runSweeps(ctx context.Context, sweeps interface {
	ReleaseExpiredReservations(context.Context, int) (int, error)
	ExpireVouchers(context.Context, int) (int, error)
}, deals interface {
	RunDealSweeps(context.Context) (store.DealSweepResult, error)
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
}
