package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	sqlc "github.com/dealo/dealo/services/api/internal/store/sqlc"
)

// DealJobs runs the deal lifecycle and counter sweeps. Every step is idempotent, so repeated or
// overlapping runs cannot corrupt state.
type DealJobs struct {
	queries *sqlc.Queries
}

// NewDealJobs wraps the pool for background deal sweeps.
func NewDealJobs(pool *pgxpool.Pool) *DealJobs {
	return &DealJobs{queries: sqlc.New(pool)}
}

// DealSweepResult counts the rows changed by one run.
type DealSweepResult struct {
	Activated, Expired, SoldOut, Reactivated, SoldCounts, DealRatings, MerchantRatings int64
}

// RunDealSweeps applies each sweep in lifecycle order and returns how many rows each one changed.
func (j *DealJobs) RunDealSweeps(ctx context.Context) (DealSweepResult, error) {
	var result DealSweepResult
	steps := []struct {
		name string
		run  func(context.Context) (int64, error)
		into *int64
	}{
		{"activate scheduled deals", j.queries.ActivateScheduledDeals, &result.Activated},
		{"expire ended deals", j.queries.ExpireEndedDeals, &result.Expired},
		{"recompute sold counts", j.queries.RecomputeDealSoldCounts, &result.SoldCounts},
		{"recompute deal ratings", j.queries.RecomputeDealRatings, &result.DealRatings},
		{"recompute merchant ratings", j.queries.RecomputeMerchantRatings, &result.MerchantRatings},
	}
	for _, step := range steps {
		changed, err := step.run(ctx)
		if err != nil {
			return result, fmt.Errorf("%s: %w", step.name, err)
		}
		*step.into = changed
	}
	return result, nil
}
