package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/grupon/grupon/services/api/internal/domain/identity"
	sqlc "github.com/grupon/grupon/services/api/internal/store/sqlc"
)

// GetUserByPhone finds an active account by its phone number.
func (r *IdentityRepository) GetUserByPhone(ctx context.Context, phone string) (identity.User, error) {
	row, err := r.queries.GetUserByPhone(ctx, pgtype.Text{String: phone, Valid: true})
	return toUserResult(row, err)
}

// CreatePhoneUser creates an account for a phone with the user role, in one transaction.
func (r *IdentityRepository) CreatePhoneUser(ctx context.Context, phone, email string) (identity.User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return identity.User{}, fmt.Errorf("begin create phone user: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	queries := r.queries.WithTx(tx)
	row, err := queries.CreatePhoneUser(ctx, sqlc.CreatePhoneUserParams{
		Email:    email,
		Phone:    pgtype.Text{String: phone, Valid: true},
		Name:     "Foydalanuvchi",
		Locale:   sqlc.LocaleCode("uz"),
		Currency: sqlc.CurrencyCode("UZS"),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return identity.User{}, identity.ErrEmailTaken
		}
		return identity.User{}, fmt.Errorf("insert phone user: %w", err)
	}
	if err := queries.CreateRoleAssignment(ctx, sqlc.CreateRoleAssignmentParams{UserID: row.ID, Role: sqlc.AppRole("user")}); err != nil {
		return identity.User{}, fmt.Errorf("grant user role: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return identity.User{}, fmt.Errorf("commit phone user: %w", err)
	}
	return toUser(row), nil
}

// InsertPhoneOTP stores the hash of a new sign-in code.
func (r *IdentityRepository) InsertPhoneOTP(ctx context.Context, phone, codeHash string, expiresAt time.Time) error {
	return r.queries.InsertPhoneOTP(ctx, sqlc.InsertPhoneOTPParams{
		Phone:     phone,
		CodeHash:  codeHash,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
}

// LatestPhoneOTP returns the newest unused code for a phone, or identity.ErrNotFound.
func (r *IdentityRepository) LatestPhoneOTP(ctx context.Context, phone string) (identity.PhoneOTP, error) {
	row, err := r.queries.LatestPhoneOTP(ctx, phone)
	if errors.Is(err, pgx.ErrNoRows) {
		return identity.PhoneOTP{}, identity.ErrNotFound
	}
	if err != nil {
		return identity.PhoneOTP{}, fmt.Errorf("latest phone code: %w", err)
	}
	return identity.PhoneOTP{
		ID:        toUUID(row.ID),
		CodeHash:  row.CodeHash,
		Attempts:  int(row.Attempts),
		ExpiresAt: row.ExpiresAt.Time,
		CreatedAt: row.CreatedAt.Time,
	}, nil
}

// IncrementPhoneOTPAttempts counts one wrong guess against a code.
func (r *IdentityRepository) IncrementPhoneOTPAttempts(ctx context.Context, id uuid.UUID) error {
	return r.queries.IncrementPhoneOTPAttempts(ctx, toPgUUID(id))
}

// ConsumePhoneOTP marks a code as used. It reports false when another request used it first.
func (r *IdentityRepository) ConsumePhoneOTP(ctx context.Context, id uuid.UUID) (bool, error) {
	rows, err := r.queries.ConsumePhoneOTP(ctx, toPgUUID(id))
	if err != nil {
		return false, fmt.Errorf("consume phone code: %w", err)
	}
	return rows == 1, nil
}

// LatestIssuedPhoneOTP returns the newest code for a phone, used or not, so the resend cooldown
// applies even after a code was consumed.
func (r *IdentityRepository) LatestIssuedPhoneOTP(ctx context.Context, phone string) (identity.PhoneOTP, error) {
	row, err := r.queries.LatestIssuedPhoneOTP(ctx, phone)
	if errors.Is(err, pgx.ErrNoRows) {
		return identity.PhoneOTP{}, identity.ErrNotFound
	}
	if err != nil {
		return identity.PhoneOTP{}, fmt.Errorf("latest issued phone code: %w", err)
	}
	return identity.PhoneOTP{
		ID:        toUUID(row.ID),
		CodeHash:  row.CodeHash,
		Attempts:  int(row.Attempts),
		ExpiresAt: row.ExpiresAt.Time,
		CreatedAt: row.CreatedAt.Time,
	}, nil
}
