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
	GetUserByEmail(ctx context.Context, email string) (User, error)
	GetUserByGoogleSub(ctx context.Context, sub string) (User, error)
	UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string) error
	MarkEmailVerified(ctx context.Context, id uuid.UUID) error
	UpdateProfile(ctx context.Context, id uuid.UUID, update ProfileUpdate) (User, error)
	LinkGoogleSub(ctx context.Context, id uuid.UUID, sub string) error
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

	CreateAuthToken(ctx context.Context, userID uuid.UUID, kind, hash string, expiresAt time.Time) error
	// ConsumeAuthToken marks an unused, unexpired token as used and returns its user. It returns
	// ErrInvalidToken when no such token exists.
	ConsumeAuthToken(ctx context.Context, kind, hash string) (uuid.UUID, error)
	InvalidateAuthTokens(ctx context.Context, userID uuid.UUID, kind string) error

	GetUserByPhone(ctx context.Context, phone string) (User, error)
	CreatePhoneUser(ctx context.Context, phone, email string) (User, error)
	InsertPhoneOTP(ctx context.Context, phone, codeHash string, expiresAt time.Time) error
	LatestPhoneOTP(ctx context.Context, phone string) (PhoneOTP, error)
	LatestIssuedPhoneOTP(ctx context.Context, phone string) (PhoneOTP, error)
	IncrementPhoneOTPAttempts(ctx context.Context, id uuid.UUID) error
	ConsumePhoneOTP(ctx context.Context, id uuid.UUID) (bool, error)
}

// RateLimiter counts attempts per key. Implementations decide the window and the limit.
type RateLimiter interface {
	Allow(ctx context.Context, key string) (bool, error)
}

// GoogleExchanger exchanges an authorization code (PKCE) for a verified Google identity.
type GoogleExchanger interface {
	Exchange(ctx context.Context, code, codeVerifier, redirectURI string) (GoogleIdentity, error)
}
