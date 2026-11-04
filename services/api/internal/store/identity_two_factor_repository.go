package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/grupon/grupon/services/api/internal/domain/identity"
)

func (r *IdentityRepository) CreateTwoFactorChallenge(ctx context.Context, userID uuid.UUID, hash string, expires time.Time) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO two_factor_challenges(token_hash,user_id,expires_at) VALUES($1,$2,$3)`, hash, userID, expires)
	return err
}

func (r *IdentityRepository) GetTwoFactorChallenge(ctx context.Context, hash string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `SELECT user_id FROM two_factor_challenges WHERE token_hash=$1 AND consumed_at IS NULL AND expires_at>now()`, hash).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, identity.ErrTwoFactorInvalid
	}
	return id, err
}

func (r *IdentityRepository) ConsumeTwoFactorChallenge(ctx context.Context, hash string) (bool, error) {
	result, err := r.pool.Exec(ctx, `UPDATE two_factor_challenges SET consumed_at=now() WHERE token_hash=$1 AND consumed_at IS NULL AND expires_at>now()`, hash)
	return result.RowsAffected() == 1, err
}

func (r *IdentityRepository) GetTwoFactor(ctx context.Context, userID uuid.UUID) (identity.TwoFactorState, error) {
	var state identity.TwoFactorState
	err := r.pool.QueryRow(ctx, `SELECT secret, enabled_at IS NOT NULL FROM user_two_factor WHERE user_id=$1`, userID).Scan(&state.Secret, &state.Enabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return state, identity.ErrNotFound
	}
	return state, err
}

func (r *IdentityRepository) BeginTwoFactorEnrollment(ctx context.Context, userID uuid.UUID, secret string) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO user_two_factor(user_id,secret,enabled_at) VALUES($1,$2,NULL)
		ON CONFLICT(user_id) DO UPDATE SET secret=excluded.secret, enabled_at=NULL`, userID, secret)
	return err
}

func (r *IdentityRepository) EnableTwoFactor(ctx context.Context, userID uuid.UUID, hashes []string, role identity.Role) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `UPDATE user_two_factor SET enabled_at=now() WHERE user_id=$1 AND enabled_at IS NULL`, userID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM two_factor_recovery_codes WHERE user_id=$1`, userID); err != nil {
		return err
	}
	for _, hash := range hashes {
		if _, err = tx.Exec(ctx, `INSERT INTO two_factor_recovery_codes(user_id,code_hash) VALUES($1,$2)`, userID, hash); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, `INSERT INTO audit_logs(actor_user_id,actor_role,action,entity_type,entity_id) VALUES($1,$2,'two_factor_enabled','user',$1)`, userID, string(role)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *IdentityRepository) DisableTwoFactor(ctx context.Context, userID uuid.UUID, role identity.Role) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `DELETE FROM two_factor_recovery_codes WHERE user_id=$1`, userID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM user_two_factor WHERE user_id=$1`, userID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO audit_logs(actor_user_id,actor_role,action,entity_type,entity_id) VALUES($1,$2,'two_factor_disabled','user',$1)`, userID, string(role)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *IdentityRepository) UseRecoveryCode(ctx context.Context, userID uuid.UUID, hash string, role identity.Role) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	result, err := tx.Exec(ctx, `UPDATE two_factor_recovery_codes SET used_at=now() WHERE user_id=$1 AND code_hash=$2 AND used_at IS NULL`, userID, hash)
	if err != nil {
		return false, err
	}
	if result.RowsAffected() != 1 {
		return false, nil
	}
	if _, err = tx.Exec(ctx, `INSERT INTO audit_logs(actor_user_id,actor_role,action,entity_type,entity_id,after_data) VALUES($1,$2,'two_factor_recovery_used','user',$1,'{}')`, userID, string(role)); err != nil {
		return false, fmt.Errorf("audit recovery use: %w", err)
	}
	return true, tx.Commit(ctx)
}
