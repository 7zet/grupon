package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/grupon/grupon/services/api/internal/domain/reviews"
	sqlc "github.com/grupon/grupon/services/api/internal/store/sqlc"
)

// ReviewRepository persists reviews and reports.
type ReviewRepository struct {
	pool *pgxpool.Pool
}

// NewReviewRepository binds review writes to a pool.
func NewReviewRepository(pool *pgxpool.Pool) *ReviewRepository {
	return &ReviewRepository{pool: pool}
}

var _ reviews.Repository = (*ReviewRepository)(nil)

// Voucher loads the owner, deal and status of a voucher.
func (r *ReviewRepository) Voucher(ctx context.Context, voucherID uuid.UUID) (reviews.VoucherOwner, error) {
	row, err := sqlc.New(r.pool).GetVoucherForReview(ctx, toPgUUID(voucherID))
	if errors.Is(err, pgx.ErrNoRows) {
		return reviews.VoucherOwner{}, reviews.ErrVoucherNotFound
	}
	if err != nil {
		return reviews.VoucherOwner{}, fmt.Errorf("get voucher for review: %w", err)
	}
	return reviews.VoucherOwner{
		UserID:     toUUID(row.UserID),
		MerchantID: toUUID(row.MerchantID),
		DealID:     toUUID(row.DealID),
		Status:     string(row.Status),
	}, nil
}

// Insert stores the review. The unique voucher id makes a second review for the same voucher fail.
func (r *ReviewRepository) Insert(ctx context.Context, owner reviews.VoucherOwner, voucherID, userID uuid.UUID, in reviews.Input) (reviews.Review, error) {
	row, err := sqlc.New(r.pool).InsertReview(ctx, sqlc.InsertReviewParams{
		DealID:     toPgUUID(owner.DealID),
		MerchantID: toPgUUID(owner.MerchantID),
		UserID:     toPgUUID(userID),
		VoucherID:  toPgUUID(voucherID),
		Rating:     int16(in.Rating),
		Text:       in.Text,
		Locale:     sqlc.LocaleCode(in.Locale),
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return reviews.Review{}, reviews.ErrAlreadyReviewed
	}
	if err != nil {
		return reviews.Review{}, fmt.Errorf("insert review: %w", err)
	}
	return reviews.Review{
		ID:         toUUID(row.ID),
		DealID:     owner.DealID,
		MerchantID: owner.MerchantID,
		UserID:     userID,
		VoucherID:  voucherID,
		Rating:     in.Rating,
		Text:       in.Text,
		Locale:     in.Locale,
		Status:     reviews.StatusPublished,
		CreatedAt:  row.CreatedAt.Time,
	}, nil
}

// Find loads a review that has not been deleted.
func (r *ReviewRepository) Find(ctx context.Context, reviewID uuid.UUID) (reviews.Review, error) {
	row, err := sqlc.New(r.pool).GetReviewRow(ctx, toPgUUID(reviewID))
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && row.DeletedAt.Valid) {
		return reviews.Review{}, reviews.ErrNotFound
	}
	if err != nil {
		return reviews.Review{}, fmt.Errorf("get review: %w", err)
	}
	return reviews.Review{
		ID:         toUUID(row.ID),
		DealID:     toUUID(row.DealID),
		MerchantID: toUUID(row.MerchantID),
		UserID:     toUUID(row.UserID),
		VoucherID:  toUUID(row.VoucherID),
		Rating:     int(row.Rating),
		Text:       row.Text,
		Locale:     string(row.Locale),
		Status:     string(row.Status),
		CreatedAt:  row.CreatedAt.Time,
	}, nil
}

// Update changes rating, text and locale of a review the user wrote.
func (r *ReviewRepository) Update(ctx context.Context, reviewID, userID uuid.UUID, in reviews.Input) (bool, error) {
	rows, err := sqlc.New(r.pool).UpdateOwnReview(ctx, sqlc.UpdateOwnReviewParams{
		ID:     toPgUUID(reviewID),
		UserID: toPgUUID(userID),
		Rating: int16(in.Rating),
		Text:   in.Text,
		Locale: sqlc.LocaleCode(in.Locale),
	})
	if err != nil {
		return false, fmt.Errorf("update review: %w", err)
	}
	return rows == 1, nil
}

// SoftDelete hides a review from the user's own list and from the public pages.
func (r *ReviewRepository) SoftDelete(ctx context.Context, reviewID, userID uuid.UUID) (bool, error) {
	rows, err := sqlc.New(r.pool).SoftDeleteOwnReview(ctx, sqlc.SoftDeleteOwnReviewParams{
		ID:     toPgUUID(reviewID),
		UserID: toPgUUID(userID),
	})
	if err != nil {
		return false, fmt.Errorf("delete review: %w", err)
	}
	return rows == 1, nil
}

// Report stores one complaint per reporter. A repeat report changes nothing.
func (r *ReviewRepository) Report(ctx context.Context, reviewID, reporterID uuid.UUID, reason string) error {
	if _, err := sqlc.New(r.pool).InsertReviewReport(ctx, sqlc.InsertReviewReportParams{
		ReviewID:       toPgUUID(reviewID),
		ReporterUserID: toPgUUID(reporterID),
		Reason:         reason,
	}); err != nil {
		return fmt.Errorf("report review: %w", err)
	}
	return nil
}
