package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/grupon/grupon/services/api/internal/domain/catalog"
	"github.com/grupon/grupon/services/api/internal/domain/moderation"
	"github.com/grupon/grupon/services/api/internal/domain/reviews"
	sqlc "github.com/grupon/grupon/services/api/internal/store/sqlc"
)

// AdminReportRepository reads review reports for moderators, closes them, and reads the audit trail.
type AdminReportRepository struct {
	pool    *pgxpool.Pool
	reviews *ReviewRepository
}

// NewAdminReportRepository binds report decisions and the audit reader to the pool.
func NewAdminReportRepository(pool *pgxpool.Pool, reviews *ReviewRepository) *AdminReportRepository {
	return &AdminReportRepository{pool: pool, reviews: reviews}
}

var _ moderation.ReportRepository = (*AdminReportRepository)(nil)

// OpenReports returns one page of pending reports with their reviews, and the total count.
func (r *AdminReportRepository) OpenReports(ctx context.Context, page catalog.Page) ([]moderation.Report, int64, error) {
	queries := sqlc.New(r.pool)
	total, err := queries.CountOpenReviewReports(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count open reports: %w", err)
	}
	rows, err := queries.ListOpenReviewReports(ctx, sqlc.ListOpenReviewReportsParams{
		PageLimit:  int32(page.Size),
		PageOffset: int32(page.Offset()),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list open reports: %w", err)
	}
	reports := make([]moderation.Report, 0, len(rows))
	for _, row := range rows {
		review, err := r.reviews.Find(ctx, toUUID(row.ReviewID))
		if errors.Is(err, reviews.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, 0, err
		}
		reports = append(reports, moderation.Report{
			ID:        toUUID(row.ID),
			Reason:    row.Reason,
			Status:    string(row.Status),
			CreatedAt: row.CreatedAt.Time,
			Review:    review,
		})
	}
	return reports, total, nil
}

// Dismiss closes a pending report and writes the audit row in the same transaction.
func (r *AdminReportRepository) Dismiss(ctx context.Context, reportID uuid.UUID, audit moderation.AuditEntry) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin report dismiss: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	queries := sqlc.New(tx)
	rows, err := queries.DismissOpenReviewReport(ctx, toPgUUID(reportID))
	if err != nil {
		return false, fmt.Errorf("dismiss report: %w", err)
	}
	if rows == 0 {
		return false, nil
	}
	if err := writeModerationAudit(ctx, queries, audit); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

// HideReview hides a live review, closes its open reports and writes the audit row in one transaction.
func (r *AdminReportRepository) HideReview(ctx context.Context, reviewID uuid.UUID, audit moderation.AuditEntry) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin review hide: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	queries := sqlc.New(tx)
	rows, err := queries.HideReviewRow(ctx, toPgUUID(reviewID))
	if err != nil {
		return false, fmt.Errorf("hide review: %w", err)
	}
	if rows == 0 {
		return false, nil
	}
	if err := queries.ActionOpenReviewReports(ctx, toPgUUID(reviewID)); err != nil {
		return false, fmt.Errorf("close review reports: %w", err)
	}
	if err := writeModerationAudit(ctx, queries, audit); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

// AuditRows returns one page of the audit trail, newest first.
func (r *AdminReportRepository) AuditRows(ctx context.Context, page catalog.Page) ([]moderation.AuditRow, int64, error) {
	queries := sqlc.New(r.pool)
	total, err := queries.CountAuditLogs(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count audit logs: %w", err)
	}
	rows, err := queries.ListAuditLogs(ctx, sqlc.ListAuditLogsParams{
		PageLimit:  int32(page.Size),
		PageOffset: int32(page.Offset()),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list audit logs: %w", err)
	}
	entries := make([]moderation.AuditRow, 0, len(rows))
	for _, row := range rows {
		var after map[string]any
		if len(row.AfterData) > 0 {
			if err := json.Unmarshal(row.AfterData, &after); err != nil {
				return nil, 0, fmt.Errorf("decode audit payload: %w", err)
			}
		}
		entries = append(entries, moderation.AuditRow{
			ID:          toUUID(row.ID),
			ActorUserID: toUUID(row.ActorUserID),
			ActorRole:   string(row.ActorRole),
			Action:      row.Action,
			EntityType:  row.EntityType,
			EntityID:    toUUID(row.EntityID),
			AfterData:   after,
			CreatedAt:   row.CreatedAt.Time,
		})
	}
	return entries, total, nil
}

func writeModerationAudit(ctx context.Context, queries *sqlc.Queries, audit moderation.AuditEntry) error {
	after, err := json.Marshal(audit.After)
	if err != nil {
		return fmt.Errorf("encode audit: %w", err)
	}
	if err := queries.InsertAuditLog(ctx, sqlc.InsertAuditLogParams{
		ActorUserID: toPgUUID(audit.ActorID),
		ActorRole:   sqlc.AppRole(audit.Role),
		Action:      audit.Action,
		EntityType:  audit.EntityType,
		EntityID:    toPgUUID(audit.EntityID),
		AfterData:   after,
	}); err != nil {
		return fmt.Errorf("write audit log: %w", err)
	}
	return nil
}

// FraudSignals returns one page of fraud signals, newest first.
func (r *AdminReportRepository) FraudSignals(ctx context.Context, page catalog.Page) ([]moderation.FraudSignal, int64, error) {
	queries := sqlc.New(r.pool)
	total, err := queries.CountFraudSignals(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count fraud signals: %w", err)
	}
	rows, err := queries.ListFraudSignals(ctx, sqlc.ListFraudSignalsParams{
		PageLimit:  int32(page.Size),
		PageOffset: int32(page.Offset()),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list fraud signals: %w", err)
	}
	signals := make([]moderation.FraudSignal, 0, len(rows))
	for _, row := range rows {
		detail := map[string]any{}
		if len(row.Detail) > 0 {
			if err := json.Unmarshal(row.Detail, &detail); err != nil {
				return nil, 0, fmt.Errorf("decode fraud detail: %w", err)
			}
		}
		signals = append(signals, moderation.FraudSignal{
			ID:            toUUID(row.ID),
			Rule:          row.Rule,
			Status:        row.Status,
			MerchantID:    toUUID(row.MerchantID),
			MerchantName:  row.MerchantName.String,
			SubjectUserID: toUUID(row.SubjectUserID),
			Detail:        detail,
			CreatedAt:     row.CreatedAt.Time,
		})
	}
	return signals, total, nil
}

// Overview returns the platform totals.
func (r *AdminReportRepository) Overview(ctx context.Context) (moderation.Overview, error) {
	row, err := sqlc.New(r.pool).AdminOverviewTotals(ctx)
	if err != nil {
		return moderation.Overview{}, fmt.Errorf("admin overview: %w", err)
	}
	return moderation.Overview{Users: row.Users, Businesses: row.Businesses, ActiveDeals: row.ActiveDeals, OrdersToday: row.OrdersToday}, nil
}
