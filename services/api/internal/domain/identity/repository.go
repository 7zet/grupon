package identity

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository is the persistence port for identity. Implementations live in internal/store.
type Repository interface {
	// CreateUser inserts an account and grants it the user role in one transaction.
	// It returns ErrEmailTaken when the email is already in use.
	CreateUser(ctx context.Context, input NewUser) (User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (User, error)
	UpdateProfile(ctx context.Context, id uuid.UUID, update ProfileUpdate) (User, error)
	TouchLastLogin(ctx context.Context, id uuid.UUID) error
	SoftDeleteUser(ctx context.Context, id uuid.UUID) error
	ListRoles(ctx context.Context, id uuid.UUID) ([]Role, error)

	CreateRefreshToken(ctx context.Context, token RefreshToken) error
	GetRefreshTokenByHash(ctx context.Context, hash string) (RefreshToken, error)
	// RotateRefreshToken revokes the current token and inserts next in one transaction. It returns
	// ErrRefreshTokenReused when the current token was already claimed by another request.
	RotateRefreshToken(ctx context.Context, currentID uuid.UUID, next RefreshToken) error
	RevokeRefreshTokenByHash(ctx context.Context, hash string) error
	RevokeRefreshFamily(ctx context.Context, familyID uuid.UUID) error
	// RevokeUserSessions revokes every refresh token of the user, except the family keep when set.
	RevokeUserSessions(ctx context.Context, id uuid.UUID, keep *uuid.UUID) error

	GetUserByPhone(ctx context.Context, phone string) (User, error)
	CreatePhoneUser(ctx context.Context, phone, email string) (User, error)
	InsertPhoneOTP(ctx context.Context, phone, codeHash string, expiresAt time.Time) error
	LatestPhoneOTP(ctx context.Context, phone string) (PhoneOTP, error)
	LatestIssuedPhoneOTP(ctx context.Context, phone string) (PhoneOTP, error)
	IncrementPhoneOTPAttempts(ctx context.Context, id uuid.UUID) error
	ConsumePhoneOTP(ctx context.Context, id uuid.UUID) (bool, error)
	IncrementSMSDailyUsage(ctx context.Context, day time.Time, scope, key string) (int, error)

	CreateTwoFactorChallenge(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error
	GetTwoFactorChallenge(ctx context.Context, tokenHash string) (uuid.UUID, error)
	ConsumeTwoFactorChallenge(ctx context.Context, tokenHash string) (bool, error)
	GetTwoFactor(ctx context.Context, userID uuid.UUID) (TwoFactorState, error)
	BeginTwoFactorEnrollment(ctx context.Context, userID uuid.UUID, secret string) error
	EnableTwoFactor(ctx context.Context, userID uuid.UUID, recoveryHashes []string, actorRole Role) error
	DisableTwoFactor(ctx context.Context, userID uuid.UUID, actorRole Role) error
	UseRecoveryCode(ctx context.Context, userID uuid.UUID, codeHash string, actorRole Role) (bool, error)
}

// RateLimiter counts attempts per key. Implementations decide the window and the limit.
type RateLimiter interface {
	Allow(ctx context.Context, key string) (bool, error)
}
