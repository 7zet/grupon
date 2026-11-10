package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dealo/dealo/services/api/internal/domain/merchant"
	sqlc "github.com/dealo/dealo/services/api/internal/store/sqlc"
)

// MerchantStaffRepository implements merchant.StaffRepository on PostgreSQL.
type MerchantStaffRepository struct {
	pool *pgxpool.Pool
}

// NewMerchantStaffRepository returns the staff store.
func NewMerchantStaffRepository(pool *pgxpool.Pool) *MerchantStaffRepository {
	return &MerchantStaffRepository{pool: pool}
}

// UserIDByPhone finds a registered user by phone, or ErrUserNotFound.
func (r *MerchantStaffRepository) UserIDByPhone(ctx context.Context, phone string) (uuid.UUID, error) {
	id, err := sqlc.New(r.pool).UserIDByPhone(ctx, pgtype.Text{String: phone, Valid: true})
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, merchant.ErrUserNotFound
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("find user by phone: %w", err)
	}
	return toUUID(id), nil
}

// List returns the staff of a business, owner first in creation order.
func (r *MerchantStaffRepository) List(ctx context.Context, merchantID uuid.UUID) ([]merchant.StaffMember, error) {
	rows, err := sqlc.New(r.pool).ListMerchantStaffRows(ctx, toPgUUID(merchantID))
	if err != nil {
		return nil, fmt.Errorf("list staff: %w", err)
	}
	out := make([]merchant.StaffMember, 0, len(rows))
	for _, row := range rows {
		out = append(out, staffFromRow(row.ID, row.UserID, row.Name, row.Email, row.Phone, row.Role, string(row.Status)))
	}
	return out, nil
}

// Add creates the role assignment and its audit row together. A second role for the same user is refused.
func (r *MerchantStaffRepository) Add(ctx context.Context, merchantID, userID uuid.UUID, role string, audit merchant.StaffAudit) (uuid.UUID, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("begin staff invite: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	queries := sqlc.New(tx)
	id, err := queries.InsertStaffAssignment(ctx, sqlc.InsertStaffAssignmentParams{
		UserID:     toPgUUID(userID),
		Role:       sqlc.AppRole(role),
		MerchantID: toPgUUID(merchantID),
	})
	if isUniqueViolation(err) {
		return uuid.Nil, merchant.ErrAlreadyStaff
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert staff: %w", err)
	}
	if err := writeStaffAudit(ctx, queries, audit); err != nil {
		return uuid.Nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("commit staff invite: %w", err)
	}
	return toUUID(id), nil
}

// Assignment returns one staff member of the business, or ErrStaffNotFound.
func (r *MerchantStaffRepository) Assignment(ctx context.Context, merchantID, assignmentID uuid.UUID) (merchant.StaffMember, error) {
	row, err := sqlc.New(r.pool).StaffAssignmentForMerchant(ctx, sqlc.StaffAssignmentForMerchantParams{
		ID:         toPgUUID(assignmentID),
		MerchantID: toPgUUID(merchantID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return merchant.StaffMember{}, merchant.ErrStaffNotFound
	}
	if err != nil {
		return merchant.StaffMember{}, fmt.Errorf("load staff: %w", err)
	}
	return merchant.StaffMember{AssignmentID: toUUID(row.ID), UserID: toUUID(row.UserID), Role: row.Role}, nil
}

// SetRole changes a manager or cashier role with its audit row.
func (r *MerchantStaffRepository) SetRole(ctx context.Context, merchantID, assignmentID uuid.UUID, role string, audit merchant.StaffAudit) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin role change: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	queries := sqlc.New(tx)
	rows, err := queries.UpdateStaffRole(ctx, sqlc.UpdateStaffRoleParams{
		ID:         toPgUUID(assignmentID),
		MerchantID: toPgUUID(merchantID),
		Role:       sqlc.AppRole(role),
	})
	if err != nil {
		return fmt.Errorf("update role: %w", err)
	}
	if rows == 0 {
		return merchant.ErrStaffNotFound
	}
	if err := writeStaffAudit(ctx, queries, audit); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Remove deletes a manager or cashier assignment with its audit row.
func (r *MerchantStaffRepository) Remove(ctx context.Context, merchantID, assignmentID uuid.UUID, audit merchant.StaffAudit) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin removal: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	queries := sqlc.New(tx)
	rows, err := queries.DeleteStaffAssignment(ctx, sqlc.DeleteStaffAssignmentParams{
		ID:         toPgUUID(assignmentID),
		MerchantID: toPgUUID(merchantID),
	})
	if err != nil {
		return fmt.Errorf("delete staff: %w", err)
	}
	if rows == 0 {
		return merchant.ErrStaffNotFound
	}
	if err := writeStaffAudit(ctx, queries, audit); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func writeStaffAudit(ctx context.Context, queries *sqlc.Queries, audit merchant.StaffAudit) error {
	after, err := json.Marshal(audit.After)
	if err != nil {
		return fmt.Errorf("encode staff audit: %w", err)
	}
	if err := queries.InsertAuditLog(ctx, sqlc.InsertAuditLogParams{
		ActorUserID: toPgUUID(audit.ActorID),
		ActorRole:   sqlc.AppRole(audit.Role),
		Action:      audit.Action,
		EntityType:  "merchant",
		EntityID:    toPgUUID(audit.MerchantID),
		AfterData:   after,
	}); err != nil {
		return fmt.Errorf("write staff audit: %w", err)
	}
	return nil
}

func staffFromRow(assignmentID, userID pgtype.UUID, name, email string, phone pgtype.Text, role, status string) merchant.StaffMember {
	return merchant.StaffMember{
		AssignmentID: toUUID(assignmentID),
		UserID:       toUUID(userID),
		Name:         name,
		Email:        email,
		Phone:        phone.String,
		Role:         role,
		Status:       status,
	}
}
