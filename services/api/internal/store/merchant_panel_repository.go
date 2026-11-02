package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/grupon/grupon/services/api/internal/domain/merchant"
	sqlc "github.com/grupon/grupon/services/api/internal/store/sqlc"
)

// MerchantPanelRepository implements merchant.PanelRepository on PostgreSQL.
type MerchantPanelRepository struct {
	pool *pgxpool.Pool
}

// NewMerchantPanelRepository returns the panel store.
func NewMerchantPanelRepository(pool *pgxpool.Pool) *MerchantPanelRepository {
	return &MerchantPanelRepository{pool: pool}
}

// Payouts lists payout runs with their gross, commission and net.
func (r *MerchantPanelRepository) Payouts(ctx context.Context, merchantID uuid.UUID) ([]merchant.PayoutRow, error) {
	rows, err := sqlc.New(r.pool).MerchantPayoutRows(ctx, toPgUUID(merchantID))
	if err != nil {
		return nil, fmt.Errorf("list payouts: %w", err)
	}
	out := make([]merchant.PayoutRow, 0, len(rows))
	for _, row := range rows {
		item := merchant.PayoutRow{
			ID:          toUUID(row.ID),
			PeriodStart: row.PeriodStart.Time,
			PeriodEnd:   row.PeriodEnd.Time,
			Gross:       row.Gross,
			Commission:  row.Commission,
			Net:         row.Net,
			Status:      payoutStatusForMerchant(row.Status),
		}
		if row.Status == "sent" || row.Status == "reconciled" {
			paid := row.PaidAt.Time
			item.PaidAt = &paid
		}
		out = append(out, item)
	}
	return out, nil
}

// payoutStatusForMerchant maps the internal run status to the three states the merchant sees.
func payoutStatusForMerchant(status string) string {
	switch status {
	case "sent", "reconciled":
		return "paid"
	case "approved":
		return "processing"
	default:
		return "pending"
	}
}

// Customers lists buyers with their order counts and totals.
func (r *MerchantPanelRepository) Customers(ctx context.Context, merchantID uuid.UUID) ([]merchant.CustomerRow, error) {
	rows, err := sqlc.New(r.pool).MerchantCustomerRows(ctx, toPgUUID(merchantID))
	if err != nil {
		return nil, fmt.Errorf("list customers: %w", err)
	}
	out := make([]merchant.CustomerRow, 0, len(rows))
	for _, row := range rows {
		customer := merchant.CustomerRow{
			UserID:        toUUID(row.UserID),
			Name:          row.Name,
			OrdersCount:   int(row.OrdersCount),
			VouchersCount: int(row.VouchersCount),
			TotalSpent:    row.TotalSpent,
		}
		if last, ok := row.LastOrderAt.(time.Time); ok {
			customer.LastOrderAt = last
		}
		out = append(out, customer)
	}
	return out, nil
}

// Analytics returns totals and the daily sales of the period.
func (r *MerchantPanelRepository) Analytics(ctx context.Context, merchantID uuid.UUID, from, to time.Time) (merchant.Analytics, error) {
	q := sqlc.New(r.pool)
	totals, err := q.MerchantSoldTotals(ctx, sqlc.MerchantSoldTotalsParams{
		MerchantID: toPgUUID(merchantID),
		FromAt:     pgtype.Timestamptz{Time: from, Valid: true},
		ToAt:       pgtype.Timestamptz{Time: to, Valid: true},
	})
	if err != nil {
		return merchant.Analytics{}, fmt.Errorf("sales totals: %w", err)
	}
	redeemed, err := q.MerchantRedeemedCount(ctx, sqlc.MerchantRedeemedCountParams{
		MerchantID: toPgUUID(merchantID),
		FromAt:     pgtype.Timestamptz{Time: from, Valid: true},
		ToAt:       pgtype.Timestamptz{Time: to, Valid: true},
	})
	if err != nil {
		return merchant.Analytics{}, fmt.Errorf("redeemed count: %w", err)
	}
	days, err := q.MerchantSalesByDay(ctx, sqlc.MerchantSalesByDayParams{
		MerchantID: toPgUUID(merchantID),
		FromAt:     pgtype.Timestamptz{Time: from, Valid: true},
		ToAt:       pgtype.Timestamptz{Time: to, Valid: true},
	})
	if err != nil {
		return merchant.Analytics{}, fmt.Errorf("sales by day: %w", err)
	}
	out := merchant.Analytics{
		GrossSales:     totals.GrossSales,
		VouchersSold:   totals.VouchersSold,
		VouchersRedeem: redeemed,
	}
	for _, day := range days {
		out.Sales = append(out.Sales, merchant.DayPoint{Date: day.Day.Time, Amount: day.Amount})
	}
	return out, nil
}

// Reviews lists reviews of the business with their replies.
func (r *MerchantPanelRepository) Reviews(ctx context.Context, merchantID uuid.UUID) ([]merchant.ReviewRow, error) {
	rows, err := sqlc.New(r.pool).MerchantReviewRows(ctx, toPgUUID(merchantID))
	if err != nil {
		return nil, fmt.Errorf("list reviews: %w", err)
	}
	out := make([]merchant.ReviewRow, 0, len(rows))
	for _, row := range rows {
		item := merchant.ReviewRow{
			ID:        toUUID(row.ID),
			DealID:    toUUID(row.DealID),
			UserID:    toUUID(row.UserID),
			VoucherID: toUUID(row.VoucherID),
			Rating:    int(row.Rating),
			Text:      row.Text,
			Locale:    string(row.Locale),
			Status:    string(row.Status),
			CreatedAt: row.CreatedAt.Time,
		}
		if row.ReplyID.Valid {
			item.Reply = &merchant.ReviewReply{
				ID:        toUUID(row.ReplyID),
				Text:      row.ReplyText.String,
				CreatedAt: row.ReplyCreatedAt.Time,
				UpdatedAt: row.ReplyUpdatedAt.Time,
			}
		}
		out = append(out, item)
	}
	return out, nil
}

// ReviewBelongs reports whether the review is one of the business's reviews.
func (r *MerchantPanelRepository) ReviewBelongs(ctx context.Context, merchantID, reviewID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM reviews WHERE id = $1 AND merchant_id = $2 AND deleted_at IS NULL)`,
		toPgUUID(reviewID), toPgUUID(merchantID)).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check review: %w", err)
	}
	return exists, nil
}

// ReplyFor returns the reply to a review, or nil when there is none.
func (r *MerchantPanelRepository) ReplyFor(ctx context.Context, reviewID uuid.UUID) (*merchant.ReviewReply, error) {
	row, err := sqlc.New(r.pool).ReviewReplyByReview(ctx, toPgUUID(reviewID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load reply: %w", err)
	}
	return &merchant.ReviewReply{ID: toUUID(row.ID), Text: row.Text, CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time}, nil
}

// InsertReply stores the first reply to a review. The owner or manager who wrote it is recorded by user id.
func (r *MerchantPanelRepository) InsertReply(ctx context.Context, reviewID, userID uuid.UUID, text string) (merchant.ReviewReply, error) {
	row, err := sqlc.New(r.pool).InsertReviewReply(ctx, sqlc.InsertReviewReplyParams{
		ReviewID:       toPgUUID(reviewID),
		MerchantUserID: toPgUUID(userID),
		Text:           text,
	})
	if err != nil {
		return merchant.ReviewReply{}, fmt.Errorf("insert reply: %w", err)
	}
	return merchant.ReviewReply{ID: toUUID(row.ID), Text: row.Text, CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time}, nil
}

// UpdateReply replaces the text of an existing reply.
func (r *MerchantPanelRepository) UpdateReply(ctx context.Context, reviewID uuid.UUID, text string) (merchant.ReviewReply, error) {
	row, err := sqlc.New(r.pool).UpdateReviewReply(ctx, sqlc.UpdateReviewReplyParams{ReviewID: toPgUUID(reviewID), Text: text})
	if err != nil {
		return merchant.ReviewReply{}, fmt.Errorf("update reply: %w", err)
	}
	return merchant.ReviewReply{ID: toUUID(row.ID), Text: row.Text, CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time}, nil
}

// Orders lists orders that contain the business's deals.
func (r *MerchantPanelRepository) Orders(ctx context.Context, merchantID uuid.UUID) ([]merchant.OrderRow, error) {
	rows, err := sqlc.New(r.pool).MerchantOrderRows(ctx, toPgUUID(merchantID))
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	out := make([]merchant.OrderRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, merchant.OrderRow{
			ID:        toUUID(row.ID),
			Number:    row.Number,
			Status:    row.Status,
			Subtotal:  row.Subtotal,
			Total:     row.Total,
			CreatedAt: row.CreatedAt.Time,
		})
	}
	return out, nil
}

// Settings reads the merchant's notification and default-branch choices.
func (r *MerchantPanelRepository) Settings(ctx context.Context, merchantID uuid.UUID) (merchant.Settings, error) {
	row, err := sqlc.New(r.pool).MerchantSettingsRow(ctx, toPgUUID(merchantID))
	if errors.Is(err, pgx.ErrNoRows) {
		return merchant.Settings{}, merchant.ErrNotFound
	}
	if err != nil {
		return merchant.Settings{}, fmt.Errorf("load settings: %w", err)
	}
	out := merchant.Settings{OrderEmails: row.OrderEmails, ReviewEmails: row.ReviewEmails, PayoutEmails: row.PayoutEmails}
	if row.DefaultLocationID.Valid {
		id := toUUID(row.DefaultLocationID)
		out.DefaultLocationID = &id
	}
	return out, nil
}

// UpdateSettings writes the choices. The default branch must belong to the business.
func (r *MerchantPanelRepository) UpdateSettings(ctx context.Context, merchantID uuid.UUID, s merchant.Settings) error {
	var location pgtype.UUID
	if s.DefaultLocationID != nil {
		location = toPgUUID(*s.DefaultLocationID)
	}
	rows, err := sqlc.New(r.pool).UpdateMerchantSettingsRow(ctx, sqlc.UpdateMerchantSettingsRowParams{
		MerchantID:        toPgUUID(merchantID),
		OrderEmails:       s.OrderEmails,
		ReviewEmails:      s.ReviewEmails,
		PayoutEmails:      s.PayoutEmails,
		DefaultLocationID: location,
	})
	if err != nil {
		return fmt.Errorf("update settings: %w", err)
	}
	if rows == 0 {
		return merchant.ErrNotFound
	}
	return nil
}
