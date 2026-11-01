package identitytest

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/grupon/grupon/services/api/internal/domain/identity"
)

type phoneCode struct {
	id        uuid.UUID
	phone     string
	codeHash  string
	attempts  int
	expiresAt time.Time
	createdAt time.Time
	consumed  bool
}

func (r *Repository) GetUserByPhone(_ context.Context, phone string) (identity.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, user := range r.users {
		if !r.deleted[user.ID] && user.Phone != nil && *user.Phone == phone {
			return user, nil
		}
	}
	return identity.User{}, identity.ErrNotFound
}

func (r *Repository) CreatePhoneUser(_ context.Context, phone, email string) (identity.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	user := identity.User{
		ID:       uuid.Must(uuid.NewV7()),
		Email:    email,
		Phone:    &phone,
		Name:     "Foydalanuvchi",
		Locale:   "uz",
		Currency: "UZS",
		Status:   identity.StatusActive,
	}
	r.users[user.ID] = user
	r.roles[user.ID] = []identity.Role{identity.RoleUser}
	return user, nil
}

func (r *Repository) InsertPhoneOTP(_ context.Context, phone, codeHash string, expiresAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.phoneCodes = append(r.phoneCodes, &phoneCode{
		id: uuid.Must(uuid.NewV7()), phone: phone, codeHash: codeHash, expiresAt: expiresAt, createdAt: r.now(),
	})
	return nil
}

func (r *Repository) LatestPhoneOTP(_ context.Context, phone string) (identity.PhoneOTP, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var latest *phoneCode
	for _, code := range r.phoneCodes {
		if code.phone == phone && !code.consumed && (latest == nil || code.createdAt.After(latest.createdAt)) {
			latest = code
		}
	}
	if latest == nil {
		return identity.PhoneOTP{}, identity.ErrNotFound
	}
	return identity.PhoneOTP{
		ID: latest.id, CodeHash: latest.codeHash, Attempts: latest.attempts,
		ExpiresAt: latest.expiresAt, CreatedAt: latest.createdAt,
	}, nil
}

func (r *Repository) IncrementPhoneOTPAttempts(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, code := range r.phoneCodes {
		if code.id == id {
			code.attempts++
		}
	}
	return nil
}

func (r *Repository) ConsumePhoneOTP(_ context.Context, id uuid.UUID) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, code := range r.phoneCodes {
		if code.id == id && !code.consumed {
			code.consumed = true
			return true, nil
		}
	}
	return false, nil
}

func (r *Repository) LatestIssuedPhoneOTP(_ context.Context, phone string) (identity.PhoneOTP, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var latest *phoneCode
	for _, code := range r.phoneCodes {
		if code.phone == phone && (latest == nil || code.createdAt.After(latest.createdAt)) {
			latest = code
		}
	}
	if latest == nil {
		return identity.PhoneOTP{}, identity.ErrNotFound
	}
	return identity.PhoneOTP{
		ID: latest.id, CodeHash: latest.codeHash, Attempts: latest.attempts,
		ExpiresAt: latest.expiresAt, CreatedAt: latest.createdAt,
	}, nil
}
