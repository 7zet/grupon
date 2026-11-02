package merchant

import (
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/google/uuid"
)

// Staff errors. The HTTP layer maps them to problem codes.
var (
	ErrUserNotFound   = errors.New("merchant: no registered user with this phone")
	ErrAlreadyStaff   = errors.New("merchant: this user already has a role in the business")
	ErrStaffNotFound  = errors.New("merchant: staff assignment not found")
	ErrOwnerProtected = errors.New("merchant: the owner's role cannot be changed here")
)

// Staff roles the owner may hand out. The owner itself is set at registration.
var inviteRoles = map[string]bool{RoleManager: true, RoleCashier: true}

// StaffMember is one person with a role in a business.
type StaffMember struct {
	AssignmentID uuid.UUID
	UserID       uuid.UUID
	Name         string
	Email        string
	Phone        string
	Role         string
	Status       string
}

// StaffAudit describes a staff change for the audit trail.
type StaffAudit struct {
	ActorID    uuid.UUID
	Role       string
	Action     string
	MerchantID uuid.UUID
	After      map[string]any
}

// StaffRepository persists staff changes. Every write also writes its audit row in the same transaction.
type StaffRepository interface {
	UserIDByPhone(ctx context.Context, phone string) (uuid.UUID, error)
	List(ctx context.Context, merchantID uuid.UUID) ([]StaffMember, error)
	Add(ctx context.Context, merchantID, userID uuid.UUID, role string, audit StaffAudit) (uuid.UUID, error)
	Assignment(ctx context.Context, merchantID, assignmentID uuid.UUID) (StaffMember, error)
	SetRole(ctx context.Context, merchantID, assignmentID uuid.UUID, role string, audit StaffAudit) error
	Remove(ctx context.Context, merchantID, assignmentID uuid.UUID, audit StaffAudit) error
}

var phoneForStaff = regexp.MustCompile(`^\+998[0-9]{9}$`)

// StaffService manages who works in a business. Only the owner changes staff (PRD §3 and FR-01.6).
type StaffService struct {
	repo StaffRepository
	now  func() time.Time
}

// NewStaffService binds the staff rules to a repository.
func NewStaffService(repo StaffRepository) *StaffService {
	return &StaffService{repo: repo, now: time.Now}
}

// List returns everyone with a role in the business. Any member may read it.
func (s *StaffService) List(ctx context.Context, merchantID uuid.UUID) ([]StaffMember, error) {
	return s.repo.List(ctx, merchantID)
}

// Invite gives an existing user a manager or cashier role. The user must have signed in once.
func (s *StaffService) Invite(ctx context.Context, actor Actor, merchantID uuid.UUID, phone, role string) (StaffMember, error) {
	if actor.Role != RoleOwner {
		return StaffMember{}, ErrForbidden
	}
	if !phoneForStaff.MatchString(phone) {
		return StaffMember{}, &InvalidInputError{Field: "phone", Code: "invalid_phone"}
	}
	if !inviteRoles[role] {
		return StaffMember{}, &InvalidInputError{Field: "role", Code: "invalid_role"}
	}
	userID, err := s.repo.UserIDByPhone(ctx, phone)
	if err != nil {
		return StaffMember{}, err
	}
	assignmentID, err := s.repo.Add(ctx, merchantID, userID, role, s.audit(actor, merchantID, "staff_invited", map[string]any{"role": role, "user_id": userID.String()}))
	if err != nil {
		return StaffMember{}, err
	}
	return s.repo.Assignment(ctx, merchantID, assignmentID)
}

// ChangeRole switches a manager or cashier to another of those two roles.
func (s *StaffService) ChangeRole(ctx context.Context, actor Actor, merchantID, assignmentID uuid.UUID, role string) (StaffMember, error) {
	if actor.Role != RoleOwner {
		return StaffMember{}, ErrForbidden
	}
	if !inviteRoles[role] {
		return StaffMember{}, &InvalidInputError{Field: "role", Code: "invalid_role"}
	}
	current, err := s.repo.Assignment(ctx, merchantID, assignmentID)
	if err != nil {
		return StaffMember{}, err
	}
	if current.Role == RoleOwner {
		return StaffMember{}, ErrOwnerProtected
	}
	if err := s.repo.SetRole(ctx, merchantID, assignmentID, role, s.audit(actor, merchantID, "staff_role_changed", map[string]any{"role": role, "assignment_id": assignmentID.String()})); err != nil {
		return StaffMember{}, err
	}
	return s.repo.Assignment(ctx, merchantID, assignmentID)
}

// Remove takes a manager or cashier out of the business. The owner cannot be removed.
func (s *StaffService) Remove(ctx context.Context, actor Actor, merchantID, assignmentID uuid.UUID) error {
	if actor.Role != RoleOwner {
		return ErrForbidden
	}
	current, err := s.repo.Assignment(ctx, merchantID, assignmentID)
	if err != nil {
		return err
	}
	if current.Role == RoleOwner {
		return ErrOwnerProtected
	}
	return s.repo.Remove(ctx, merchantID, assignmentID, s.audit(actor, merchantID, "staff_removed", map[string]any{"assignment_id": assignmentID.String()}))
}

func (s *StaffService) audit(actor Actor, merchantID uuid.UUID, action string, after map[string]any) StaffAudit {
	return StaffAudit{ActorID: actor.UserID, Role: actor.Role, Action: action, MerchantID: merchantID, After: after}
}
