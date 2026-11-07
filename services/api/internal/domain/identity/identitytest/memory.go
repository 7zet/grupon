// Package identitytest provides in-memory test doubles for the identity ports.
package identitytest

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/grupon/grupon/services/api/internal/domain/identity"
	"github.com/grupon/grupon/services/api/internal/platform/mail"
)

type authToken struct {
	userID    uuid.UUID
	kind      string
	expiresAt time.Time
	used      bool
}

// Repository is an in-memory identity.Repository with the same uniqueness and single-use rules as
// the PostgreSQL schema.
type Repository struct {
	mu         sync.Mutex
	now        func() time.Time
	users      map[uuid.UUID]identity.User
	deleted    map[uuid.UUID]bool
	roles      map[uuid.UUID][]identity.Role
	refresh    map[string]identity.RefreshToken
	authTokens map[string]*authToken
	phoneCodes []*phoneCode
	smsUsage   map[string]int
	twoFactor  map[uuid.UUID]identity.TwoFactorState
	challenges map[string]*authToken
	recovery   map[uuid.UUID]map[string]bool
}

var _ identity.Repository = (*Repository)(nil)

// NewRepository returns an empty repository. A nil clock uses time.Now.
func NewRepository(now func() time.Time) *Repository {
	if now == nil {
		now = time.Now
	}
	return &Repository{
		now:        now,
		users:      map[uuid.UUID]identity.User{},
		deleted:    map[uuid.UUID]bool{},
		roles:      map[uuid.UUID][]identity.Role{},
		refresh:    map[string]identity.RefreshToken{},
		authTokens: map[string]*authToken{},
		twoFactor:  map[uuid.UUID]identity.TwoFactorState{}, challenges: map[string]*authToken{}, recovery: map[uuid.UUID]map[string]bool{},
	}
}

func (r *Repository) CreateTwoFactorChallenge(_ context.Context, id uuid.UUID, hash string, expires time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.challenges[hash] = &authToken{userID: id, expiresAt: expires}
	return nil
}
func (r *Repository) GetTwoFactorChallenge(_ context.Context, hash string) (uuid.UUID, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.challenges[hash]
	if !ok || v.used || !r.now().Before(v.expiresAt) {
		return uuid.Nil, identity.ErrTwoFactorInvalid
	}
	return v.userID, nil
}
func (r *Repository) ConsumeTwoFactorChallenge(_ context.Context, hash string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.challenges[hash]
	if !ok || v.used || !r.now().Before(v.expiresAt) {
		return false, nil
	}
	v.used = true
	return true, nil
}
func (r *Repository) GetTwoFactor(_ context.Context, id uuid.UUID) (identity.TwoFactorState, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.twoFactor[id]
	if !ok {
		return v, identity.ErrNotFound
	}
	return v, nil
}
func (r *Repository) BeginTwoFactorEnrollment(_ context.Context, id uuid.UUID, secret string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.twoFactor[id] = identity.TwoFactorState{Secret: secret}
	return nil
}
func (r *Repository) EnableTwoFactor(_ context.Context, id uuid.UUID, hashes []string, _ identity.Role) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	v := r.twoFactor[id]
	v.Enabled = true
	r.twoFactor[id] = v
	r.recovery[id] = map[string]bool{}
	for _, h := range hashes {
		r.recovery[id][h] = false
	}
	return nil
}
func (r *Repository) DisableTwoFactor(_ context.Context, id uuid.UUID, _ identity.Role) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.twoFactor, id)
	delete(r.recovery, id)
	return nil
}
func (r *Repository) UseRecoveryCode(_ context.Context, id uuid.UUID, hash string, _ identity.Role) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	used, ok := r.recovery[id][hash]
	if !ok || used {
		return false, nil
	}
	r.recovery[id][hash] = true
	return true, nil
}

// GrantRole adds a role to a user, as a superadmin would.
func (r *Repository) GrantRole(userID uuid.UUID, role identity.Role) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.roles[userID] = append(r.roles[userID], role)
}

// SetStatus changes the account status, for blocking tests.
func (r *Repository) SetStatus(userID uuid.UUID, status string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	user := r.users[userID]
	user.Status = status
	r.users[userID] = user
}

// ActiveRefreshCount returns the number of unrevoked refresh tokens of a user.
func (r *Repository) ActiveRefreshCount(userID uuid.UUID) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for _, token := range r.refresh {
		if token.UserID == userID && token.RevokedAt == nil {
			count++
		}
	}
	return count
}

func (r *Repository) CreateUser(_ context.Context, input identity.NewUser) (identity.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, user := range r.users {
		if !r.deleted[user.ID] && user.Email == input.Email {
			return identity.User{}, identity.ErrEmailTaken
		}
	}
	user := identity.User{
		ID:       uuid.Must(uuid.NewV7()),
		Email:    input.Email,
		Name:     input.Name,
		Locale:   input.Locale,
		Currency: input.Currency,
		Status:   identity.StatusActive,
	}
	if input.EmailVerified {
		now := r.now()
		user.EmailVerifiedAt = &now
	}
	r.users[user.ID] = user
	r.roles[user.ID] = []identity.Role{identity.RoleUser}
	return user, nil
}

func (r *Repository) GetUserByID(_ context.Context, id uuid.UUID) (identity.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	user, ok := r.users[id]
	if !ok || r.deleted[id] {
		return identity.User{}, identity.ErrNotFound
	}
	return user, nil
}

func (r *Repository) GetUserByEmail(_ context.Context, email string) (identity.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, user := range r.users {
		if !r.deleted[id] && user.Email == email {
			return user, nil
		}
	}
	return identity.User{}, identity.ErrNotFound
}

func (r *Repository) MarkEmailVerified(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	user := r.users[id]
	if user.EmailVerifiedAt == nil {
		now := r.now()
		user.EmailVerifiedAt = &now
	}
	r.users[id] = user
	return nil
}

func (r *Repository) UpdateProfile(_ context.Context, id uuid.UUID, update identity.ProfileUpdate) (identity.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	user := r.users[id]
	if update.Name != nil {
		user.Name = *update.Name
	}
	if update.Phone != nil {
		phone := *update.Phone
		if phone == "" {
			user.Phone = nil
		} else {
			user.Phone = &phone
		}
	}
	if update.Locale != nil {
		user.Locale = *update.Locale
	}
	if update.Currency != nil {
		user.Currency = *update.Currency
	}
	r.users[id] = user
	return user, nil
}

func (r *Repository) TouchLastLogin(context.Context, uuid.UUID) error { return nil }

func (r *Repository) SoftDeleteUser(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.deleted[id] = true
	return nil
}

func (r *Repository) ListRoles(_ context.Context, id uuid.UUID) ([]identity.Role, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]identity.Role(nil), r.roles[id]...), nil
}

func (r *Repository) CreateRefreshToken(_ context.Context, token identity.RefreshToken) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	token.ID = uuid.Must(uuid.NewV7())
	r.refresh[token.TokenHash] = token
	return nil
}

func (r *Repository) GetRefreshTokenByHash(_ context.Context, hash string) (identity.RefreshToken, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	token, ok := r.refresh[hash]
	if !ok {
		return identity.RefreshToken{}, identity.ErrNotFound
	}
	return token, nil
}

func (r *Repository) RotateRefreshToken(_ context.Context, currentID uuid.UUID, next identity.RefreshToken) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for hash, token := range r.refresh {
		if token.ID != currentID {
			continue
		}
		if token.RevokedAt != nil {
			return identity.ErrRefreshTokenReused
		}
		now := r.now()
		token.RevokedAt = &now
		r.refresh[hash] = token
		next.ID = uuid.Must(uuid.NewV7())
		r.refresh[next.TokenHash] = next
		return nil
	}
	return identity.ErrNotFound
}

func (r *Repository) RevokeRefreshTokenByHash(_ context.Context, hash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if token, ok := r.refresh[hash]; ok && token.RevokedAt == nil {
		now := r.now()
		token.RevokedAt = &now
		r.refresh[hash] = token
	}
	return nil
}

func (r *Repository) RevokeRefreshFamily(_ context.Context, familyID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := r.now()
	for hash, token := range r.refresh {
		if token.FamilyID == familyID && token.RevokedAt == nil {
			token.RevokedAt = &now
			r.refresh[hash] = token
		}
	}
	return nil
}

func (r *Repository) RevokeUserSessions(_ context.Context, id uuid.UUID, keep *uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := r.now()
	for hash, token := range r.refresh {
		if token.UserID != id || token.RevokedAt != nil {
			continue
		}
		if keep != nil && token.FamilyID == *keep {
			continue
		}
		token.RevokedAt = &now
		r.refresh[hash] = token
	}
	return nil
}

func (r *Repository) CreateAuthToken(_ context.Context, userID uuid.UUID, kind, hash string, expiresAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.authTokens[hash] = &authToken{userID: userID, kind: kind, expiresAt: expiresAt}
	return nil
}

func (r *Repository) ConsumeAuthToken(_ context.Context, kind, hash string) (uuid.UUID, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	token, ok := r.authTokens[hash]
	if !ok || token.kind != kind || token.used || !r.now().Before(token.expiresAt) {
		return uuid.Nil, identity.ErrInvalidToken
	}
	token.used = true
	return token.userID, nil
}

func (r *Repository) InvalidateAuthTokens(_ context.Context, userID uuid.UUID, kind string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, token := range r.authTokens {
		if token.userID == userID && token.kind == kind {
			token.used = true
		}
	}
	return nil
}

// Mailer records every message instead of sending it.
type Mailer struct {
	mu       sync.Mutex
	messages []mail.Message
}

var _ mail.Sender = (*Mailer)(nil)

// Send stores the message.
func (m *Mailer) Send(_ context.Context, message mail.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, message)
	return nil
}

// Messages returns a copy of the recorded messages.
func (m *Mailer) Messages() []mail.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]mail.Message(nil), m.messages...)
}

// Limiter allows at most Max attempts per key and ignores time windows.
type Limiter struct {
	Max int

	mu     sync.Mutex
	counts map[string]int
}

// Allow counts the attempt and reports whether it is within the limit.
func (l *Limiter) Allow(_ context.Context, key string) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.counts == nil {
		l.counts = map[string]int{}
	}
	l.counts[key]++
	return l.counts[key] <= l.Max, nil
}
