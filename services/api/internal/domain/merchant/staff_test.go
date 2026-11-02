package merchant

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

// memStaff keeps assignments in memory so the rules can be tested without a database.
type memStaff struct {
	users    map[string]uuid.UUID
	members  map[uuid.UUID]StaffMember
	audited  []string
	ownerID  uuid.UUID
	ownerRow uuid.UUID
}

func (m *memStaff) UserIDByPhone(_ context.Context, phone string) (uuid.UUID, error) {
	id, ok := m.users[phone]
	if !ok {
		return uuid.Nil, ErrUserNotFound
	}
	return id, nil
}

func (m *memStaff) List(context.Context, uuid.UUID) ([]StaffMember, error) {
	out := make([]StaffMember, 0, len(m.members))
	for _, member := range m.members {
		out = append(out, member)
	}
	return out, nil
}

func (m *memStaff) Add(_ context.Context, _, userID uuid.UUID, role string, audit StaffAudit) (uuid.UUID, error) {
	for _, member := range m.members {
		if member.UserID == userID {
			return uuid.Nil, ErrAlreadyStaff
		}
	}
	id := uuid.New()
	m.members[id] = StaffMember{AssignmentID: id, UserID: userID, Role: role}
	m.audited = append(m.audited, audit.Action)
	return id, nil
}

func (m *memStaff) Assignment(_ context.Context, _, assignmentID uuid.UUID) (StaffMember, error) {
	member, ok := m.members[assignmentID]
	if !ok {
		return StaffMember{}, ErrStaffNotFound
	}
	return member, nil
}

func (m *memStaff) SetRole(_ context.Context, _, assignmentID uuid.UUID, role string, audit StaffAudit) error {
	member, ok := m.members[assignmentID]
	if !ok {
		return ErrStaffNotFound
	}
	member.Role = role
	m.members[assignmentID] = member
	m.audited = append(m.audited, audit.Action)
	return nil
}

func (m *memStaff) Remove(_ context.Context, _, assignmentID uuid.UUID, audit StaffAudit) error {
	if _, ok := m.members[assignmentID]; !ok {
		return ErrStaffNotFound
	}
	delete(m.members, assignmentID)
	m.audited = append(m.audited, audit.Action)
	return nil
}

func TestOnlyTheOwnerChangesStaffAndEveryChangeIsAudited(t *testing.T) {
	t.Parallel()
	cashierUser := uuid.New()
	repo := &memStaff{
		users:   map[string]uuid.UUID{"+998901110099": cashierUser},
		members: map[uuid.UUID]StaffMember{},
	}
	service := NewStaffService(repo)
	ctx := context.Background()
	merchantID := uuid.New()
	owner := Actor{UserID: uuid.New(), Role: RoleOwner}
	manager := Actor{UserID: uuid.New(), Role: RoleManager}

	if _, err := service.Invite(ctx, manager, merchantID, "+998901110099", RoleCashier); !errors.Is(err, ErrForbidden) {
		t.Fatalf("manager invite: got %v, want ErrForbidden", err)
	}
	member, err := service.Invite(ctx, owner, merchantID, "+998901110099", RoleCashier)
	if err != nil || member.Role != RoleCashier {
		t.Fatalf("owner invite: member=%+v err=%v", member, err)
	}
	if _, err := service.Invite(ctx, owner, merchantID, "+998901110099", RoleManager); !errors.Is(err, ErrAlreadyStaff) {
		t.Fatalf("second role for the same user: got %v, want ErrAlreadyStaff", err)
	}
	if _, err := service.Invite(ctx, owner, merchantID, "+998901110000", RoleCashier); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("unknown phone: got %v, want ErrUserNotFound", err)
	}
	if _, err := service.Invite(ctx, owner, merchantID, "call me", RoleCashier); err == nil {
		t.Fatal("malformed phone was accepted")
	}
	if _, err := service.Invite(ctx, owner, merchantID, "+998901110099", RoleOwner); err == nil {
		t.Fatal("owner role could be handed out by invitation")
	}

	ownerRow := uuid.New()
	repo.members[ownerRow] = StaffMember{AssignmentID: ownerRow, UserID: owner.UserID, Role: RoleOwner}
	if _, err := service.ChangeRole(ctx, owner, merchantID, ownerRow, RoleCashier); !errors.Is(err, ErrOwnerProtected) {
		t.Fatalf("owner demoted: got %v, want ErrOwnerProtected", err)
	}
	if err := service.Remove(ctx, owner, merchantID, ownerRow); !errors.Is(err, ErrOwnerProtected) {
		t.Fatalf("owner removed: got %v, want ErrOwnerProtected", err)
	}
	if err := service.Remove(ctx, owner, merchantID, member.AssignmentID); err != nil {
		t.Fatalf("remove cashier: %v", err)
	}
	if len(repo.audited) != 2 || repo.audited[0] != "staff_invited" || repo.audited[1] != "staff_removed" {
		t.Fatalf("audit trail = %v", repo.audited)
	}
}
