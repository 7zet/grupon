// Package reviews lets buyers review a deal after redeeming their voucher, and lets anyone report an abusive review.
package reviews

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Review statuses mirror the review_status enum.
const (
	StatusPublished = "published"
	StatusHidden    = "hidden"
	StatusPending   = "pending"
)

// Errors returned by the review rules. The HTTP layer maps them to problem codes.
var (
	ErrNotFound        = errors.New("reviews: review not found")
	ErrVoucherNotFound = errors.New("reviews: voucher not found")
	ErrNotRedeemed     = errors.New("reviews: voucher is not redeemed by this user")
	ErrAlreadyReviewed = errors.New("reviews: voucher already has a review")
	ErrOwnReport       = errors.New("reviews: cannot report your own review")
)

// InvalidInputError reports a field that breaks a rule the contract cannot express.
type InvalidInputError struct {
	Field string
	Code  string
}

func (e *InvalidInputError) Error() string { return "reviews: invalid " + e.Field + ": " + e.Code }

var locales = map[string]bool{"uz": true, "ru": true, "en": true}

// Input is the editable part of a review.
type Input struct {
	Rating int
	Text   string
	Locale string
}

// Review is a stored review.
type Review struct {
	ID         uuid.UUID
	DealID     uuid.UUID
	MerchantID uuid.UUID
	UserID     uuid.UUID
	VoucherID  uuid.UUID
	Rating     int
	Text       string
	Locale     string
	Status     string
	CreatedAt  time.Time
}

// VoucherOwner is the part of a voucher that decides whether a review may be written.
type VoucherOwner struct {
	UserID     uuid.UUID
	MerchantID uuid.UUID
	DealID     uuid.UUID
	Status     string
}

// Repository persists reviews and reports.
type Repository interface {
	Voucher(ctx context.Context, voucherID uuid.UUID) (VoucherOwner, error)
	Insert(ctx context.Context, owner VoucherOwner, voucherID, userID uuid.UUID, in Input) (Review, error)
	Find(ctx context.Context, reviewID uuid.UUID) (Review, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]Review, error)
	Update(ctx context.Context, reviewID, userID uuid.UUID, in Input) (bool, error)
	SoftDelete(ctx context.Context, reviewID, userID uuid.UUID) (bool, error)
	Report(ctx context.Context, reviewID, reporterID uuid.UUID, reason string) error
}

// Service applies the review rules.
type Service struct {
	repo Repository
}

// NewService binds the review rules to a repository.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Create writes the one review allowed per redeemed voucher.
func (s *Service) Create(ctx context.Context, userID, voucherID uuid.UUID, in Input) (Review, error) {
	in, err := normalize(in)
	if err != nil {
		return Review{}, err
	}
	owner, err := s.repo.Voucher(ctx, voucherID)
	if err != nil {
		return Review{}, err
	}
	if owner.UserID != userID || owner.Status != "redeemed" {
		return Review{}, ErrNotRedeemed
	}
	return s.repo.Insert(ctx, owner, voucherID, userID, in)
}

// ListOwn returns the reviews the user wrote, newest first. Deleted reviews are not listed.
func (s *Service) ListOwn(ctx context.Context, userID uuid.UUID) ([]Review, error) {
	return s.repo.ListByUser(ctx, userID)
}

// Update changes the text or rating of a review the user wrote.
func (s *Service) Update(ctx context.Context, userID, reviewID uuid.UUID, in Input) (Review, error) {
	in, err := normalize(in)
	if err != nil {
		return Review{}, err
	}
	updated, err := s.repo.Update(ctx, reviewID, userID, in)
	if err != nil {
		return Review{}, err
	}
	if !updated {
		return Review{}, ErrNotFound
	}
	return s.repo.Find(ctx, reviewID)
}

// Delete soft-deletes a review the user wrote.
func (s *Service) Delete(ctx context.Context, userID, reviewID uuid.UUID) error {
	deleted, err := s.repo.SoftDelete(ctx, reviewID, userID)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrNotFound
	}
	return nil
}

// Report records a complaint once per reporter. Repeat reports are accepted without a second row.
func (s *Service) Report(ctx context.Context, reporterID, reviewID uuid.UUID, reason string) error {
	reason = strings.TrimSpace(reason)
	if n := len([]rune(reason)); n < 10 || n > 500 {
		return &InvalidInputError{Field: "reason", Code: "length"}
	}
	review, err := s.repo.Find(ctx, reviewID)
	if err != nil {
		return err
	}
	if review.UserID == reporterID {
		return ErrOwnReport
	}
	return s.repo.Report(ctx, reviewID, reporterID, reason)
}

func normalize(in Input) (Input, error) {
	in.Text = strings.TrimSpace(in.Text)
	if in.Rating < 1 || in.Rating > 5 {
		return Input{}, &InvalidInputError{Field: "rating", Code: "range"}
	}
	if n := len([]rune(in.Text)); n < 20 || n > 2000 {
		return Input{}, &InvalidInputError{Field: "text", Code: "length"}
	}
	if !locales[in.Locale] {
		return Input{}, &InvalidInputError{Field: "locale", Code: "invalid_locale"}
	}
	return in, nil
}
