package commerce

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/grupon/grupon/services/api/internal/domain/catalog"
)

// RecordView notes that the user opened a deal. The store keeps only the most recent entries.
func (s *Service) RecordView(ctx context.Context, userID, dealID uuid.UUID) error {
	if err := s.requireDeal(ctx, dealID); err != nil {
		return err
	}
	return s.repo.RecordView(ctx, userID, dealID)
}

// RecentlyViewed returns the user's recently viewed deals that can still be shown.
func (s *Service) RecentlyViewed(ctx context.Context, userID uuid.UUID) ([]catalog.DealSummary, error) {
	ids, err := s.repo.RecentDealIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.publishedDeals(ctx, ids)
}

func (s *Service) requireDeal(ctx context.Context, dealID uuid.UUID) error {
	summaries, err := s.deals.DealSummaries(ctx, []uuid.UUID{dealID})
	if err != nil {
		return fmt.Errorf("load deal: %w", err)
	}
	if len(summaries) != 1 {
		return ErrDealNotFound
	}
	return nil
}

// publishedDeals keeps the order of ids and drops deals that are no longer publishable.
func (s *Service) publishedDeals(ctx context.Context, ids []uuid.UUID) ([]catalog.DealSummary, error) {
	if len(ids) == 0 {
		return []catalog.DealSummary{}, nil
	}
	summaries, err := s.deals.DealSummaries(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("load deals: %w", err)
	}
	byID := make(map[uuid.UUID]catalog.DealSummary, len(summaries))
	for _, summary := range summaries {
		byID[summary.ID] = summary
	}
	now := s.now()
	result := make([]catalog.DealSummary, 0, len(ids))
	for _, id := range ids {
		if summary, ok := byID[id]; ok && catalog.IsPublishable(summary, now) {
			result = append(result, summary)
		}
	}
	return result, nil
}
