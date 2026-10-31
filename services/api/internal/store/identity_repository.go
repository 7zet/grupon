package store

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/grupon/grupon/services/api/internal/domain/identity"
	sqlc "github.com/grupon/grupon/services/api/internal/store/sqlc"
)

const uniqueViolation = "23505"

// IdentityRepository implements identity.Repository on top of sqlc queries.
type IdentityRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewIdentityRepository binds the repository to a connection pool. Multi-step writes use transactions.
func NewIdentityRepository(pool *pgxpool.Pool) *IdentityRepository {
	return &IdentityRepository{pool: pool, queries: sqlc.New(pool)}
}

var _ identity.Repository = (*IdentityRepository)(nil)

func (r *IdentityRepository) CreateUser(ctx context.Context, input identity.NewUser) (identity.User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return identity.User{}, fmt.Errorf("begin create user: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	queries := r.queries.WithTx(tx)
	var verifiedAt pgtype.Timestamptz
	if input.EmailVerified {
		verifiedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
	}
	row, err := queries.CreateUser(ctx, sqlc.CreateUserParams{
		Email:           input.Email,
		PasswordHash:    optionalText(input.PasswordHash),
		GoogleSub:       optionalText(input.GoogleSub),
		Name:            input.Name,
		Locale:          sqlc.LocaleCode(input.Locale),
		Currency:        sqlc.CurrencyCode(input.Currency),
		EmailVerifiedAt: verifiedAt,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return identity.User{}, identity.ErrEmailTaken
		}
		return identity.User{}, fmt.Errorf("insert user: %w", err)
	}
	if err := queries.CreateRoleAssignment(ctx, sqlc.CreateRoleAssignmentParams{
		UserID:     row.ID,
		Role:       sqlc.AppRole(identity.RoleUser),
		MerchantID: pgtype.UUID{},
	}); err != nil {
		return identity.User{}, fmt.Errorf("grant user role: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return identity.User{}, fmt.Errorf("commit create user: %w", err)
	}
	return toUser(row), nil
}

func (r *IdentityRepository) GetUserByID(ctx context.Context, id uuid.UUID) (identity.User, error) {
	row, err := r.queries.GetUserByID(ctx, toPgUUID(id))
	return toUserResult(row, err)
}

func (r *IdentityRepository) GetUserByEmail(ctx context.Context, email string) (identity.User, error) {
	row, err := r.queries.GetUserByEmail(ctx, email)
	return toUserResult(row, err)
}

func (r *IdentityRepository) GetUserByGoogleSub(ctx context.Context, sub string) (identity.User, error) {
	row, err := r.queries.GetUserByGoogleSub(ctx, pgtype.Text{String: sub, Valid: true})
	return toUserResult(row, err)
}

func (r *IdentityRepository) UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string) error {
	return r.queries.UpdateUserPasswordHash(ctx, sqlc.UpdateUserPasswordHashParams{
		ID:           toPgUUID(id),
		PasswordHash: pgtype.Text{String: hash, Valid: true},
	})
}

func (r *IdentityRepository) MarkEmailVerified(ctx context.Context, id uuid.UUID) error {
	return r.queries.MarkUserEmailVerified(ctx, toPgUUID(id))
}

func (r *IdentityRepository) UpdateProfile(ctx context.Context, id uuid.UUID, update identity.ProfileUpdate) (identity.User, error) {
	current, err := r.queries.GetUserByID(ctx, toPgUUID(id))
	if err != nil {
		return identity.User{}, mapNotFound(err)
	}
	merged := sqlc.UpdateUserProfileParams{
		ID:       toPgUUID(id),
		Name:     current.Name,
		Phone:    current.Phone,
		Locale:   current.Locale,
		Currency: current.Currency,
	}
	if update.Name != nil {
		merged.Name = *update.Name
	}
	if update.Phone != nil {
		merged.Phone = pgtype.Text{String: *update.Phone, Valid: *update.Phone != ""}
	}
	if update.Locale != nil {
		merged.Locale = sqlc.LocaleCode(*update.Locale)
	}
	if update.Currency != nil {
		merged.Currency = sqlc.CurrencyCode(*update.Currency)
	}
	row, err := r.queries.UpdateUserProfile(ctx, merged)
	return toUserResult(row, err)
}

func (r *IdentityRepository) LinkGoogleSub(ctx context.Context, id uuid.UUID, sub string) error {
	return r.queries.LinkUserGoogleSub(ctx, sqlc.LinkUserGoogleSubParams{
		ID:        toPgUUID(id),
		GoogleSub: pgtype.Text{String: sub, Valid: true},
	})
}

func (r *IdentityRepository) TouchLastLogin(ctx context.Context, id uuid.UUID) error {
	return r.queries.TouchUserLastLogin(ctx, toPgUUID(id))
}

func (r *IdentityRepository) SoftDeleteUser(ctx context.Context, id uuid.UUID) error {
	return r.queries.SoftDeleteUser(ctx, toPgUUID(id))
}

func (r *IdentityRepository) ListRoles(ctx context.Context, id uuid.UUID) ([]identity.Role, error) {
	rows, err := r.queries.ListUserRoles(ctx, toPgUUID(id))
	if err != nil {
		return nil, fmt.Errorf("list user roles: %w", err)
	}
	roles := make([]identity.Role, 0, len(rows))
	for _, row := range rows {
		roles = append(roles, identity.Role(row.Role))
	}
	return roles, nil
}

func (r *IdentityRepository) CreateRefreshToken(ctx context.Context, token identity.RefreshToken) error {
	return r.queries.CreateRefreshToken(ctx, createRefreshParams(token))
}

func (r *IdentityRepository) GetRefreshTokenByHash(ctx context.Context, hash string) (identity.RefreshToken, error) {
	row, err := r.queries.GetRefreshTokenByHash(ctx, hash)
	if err != nil {
		return identity.RefreshToken{}, mapNotFound(err)
	}
	token := identity.RefreshToken{
		ID:        toUUID(row.ID),
		UserID:    toUUID(row.UserID),
		TokenHash: row.TokenHash,
		FamilyID:  toUUID(row.FamilyID),
		ExpiresAt: row.ExpiresAt.Time,
		UserAgent: row.UserAgent.String,
	}
	if row.RevokedAt.Valid {
		revoked := row.RevokedAt.Time
		token.RevokedAt = &revoked
	}
	if row.Ip != nil {
		token.IP = row.Ip.String()
	}
	return token, nil
}

// RotateRefreshToken claims the current token and inserts its successor atomically. A token that
// is already revoked cannot be claimed, which is how concurrent reuse is detected.
func (r *IdentityRepository) RotateRefreshToken(ctx context.Context, currentID uuid.UUID, next identity.RefreshToken) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin rotate refresh token: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	queries := r.queries.WithTx(tx)
	if _, err := queries.ClaimRefreshToken(ctx, toPgUUID(currentID)); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return identity.ErrRefreshTokenReused
		}
		return fmt.Errorf("claim refresh token: %w", err)
	}
	if err := queries.CreateRefreshToken(ctx, createRefreshParams(next)); err != nil {
		return fmt.Errorf("insert rotated refresh token: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit rotate refresh token: %w", err)
	}
	return nil
}

func (r *IdentityRepository) RevokeRefreshTokenByHash(ctx context.Context, hash string) error {
	return r.queries.RevokeRefreshTokenByHash(ctx, hash)
}

func (r *IdentityRepository) RevokeRefreshFamily(ctx context.Context, familyID uuid.UUID) error {
	return r.queries.RevokeRefreshFamily(ctx, toPgUUID(familyID))
}

func (r *IdentityRepository) RevokeUserSessions(ctx context.Context, id uuid.UUID, keep *uuid.UUID) error {
	if keep == nil {
		return r.queries.RevokeAllUserRefreshTokens(ctx, toPgUUID(id))
	}
	return r.queries.RevokeUserRefreshTokensExceptFamily(ctx, sqlc.RevokeUserRefreshTokensExceptFamilyParams{
		UserID:       toPgUUID(id),
		KeepFamilyID: toPgUUID(*keep),
	})
}

func (r *IdentityRepository) CreateAuthToken(ctx context.Context, userID uuid.UUID, kind, hash string, expiresAt time.Time) error {
	return r.queries.CreateAuthToken(ctx, sqlc.CreateAuthTokenParams{
		UserID:    toPgUUID(userID),
		Kind:      kind,
		TokenHash: hash,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
}

func (r *IdentityRepository) ConsumeAuthToken(ctx context.Context, kind, hash string) (uuid.UUID, error) {
	id, err := r.queries.ConsumeAuthToken(ctx, sqlc.ConsumeAuthTokenParams{Kind: kind, TokenHash: hash})
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, identity.ErrInvalidToken
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("consume auth token: %w", err)
	}
	return toUUID(id), nil
}

func (r *IdentityRepository) InvalidateAuthTokens(ctx context.Context, userID uuid.UUID, kind string) error {
	return r.queries.InvalidateUserAuthTokens(ctx, sqlc.InvalidateUserAuthTokensParams{
		UserID: toPgUUID(userID),
		Kind:   kind,
	})
}

func createRefreshParams(token identity.RefreshToken) sqlc.CreateRefreshTokenParams {
	var ip *netip.Addr
	if parsed, err := netip.ParseAddr(token.IP); err == nil {
		ip = &parsed
	}
	return sqlc.CreateRefreshTokenParams{
		UserID:    toPgUUID(token.UserID),
		TokenHash: token.TokenHash,
		FamilyID:  toPgUUID(token.FamilyID),
		ExpiresAt: pgtype.Timestamptz{Time: token.ExpiresAt, Valid: true},
		UserAgent: optionalText(&token.UserAgent),
		Ip:        ip,
	}
}

func toUserResult(row sqlc.User, err error) (identity.User, error) {
	if err != nil {
		return identity.User{}, mapNotFound(err)
	}
	return toUser(row), nil
}

func toUser(row sqlc.User) identity.User {
	user := identity.User{
		ID:       toUUID(row.ID),
		Email:    row.Email,
		Name:     row.Name,
		Locale:   string(row.Locale),
		Currency: string(row.Currency),
		Status:   string(row.Status),
	}
	if row.PasswordHash.Valid {
		value := row.PasswordHash.String
		user.PasswordHash = &value
	}
	if row.GoogleSub.Valid {
		value := row.GoogleSub.String
		user.GoogleSub = &value
	}
	if row.Phone.Valid {
		value := row.Phone.String
		user.Phone = &value
	}
	if row.AvatarUrl.Valid {
		value := row.AvatarUrl.String
		user.AvatarURL = &value
	}
	if row.EmailVerifiedAt.Valid {
		value := row.EmailVerifiedAt.Time
		user.EmailVerifiedAt = &value
	}
	return user
}

func mapNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return identity.ErrNotFound
	}
	return fmt.Errorf("identity query: %w", err)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == uniqueViolation
}

func optionalText(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *value, Valid: true}
}
