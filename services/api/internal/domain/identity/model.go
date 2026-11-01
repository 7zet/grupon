package identity

import (
	"time"

	"github.com/google/uuid"
)

// Role is an application role. Merchant roles are scoped to a merchant in role assignments.
type Role string

// Roles known to the platform. Only RoleUser is granted automatically.
const (
	RoleUser            Role = "user"
	RoleMerchantOwner   Role = "merchant_owner"
	RoleMerchantManager Role = "merchant_manager"
	RoleMerchantCashier Role = "merchant_cashier"
	RoleAdmin           Role = "admin"
	RoleModerator       Role = "moderator"
	RoleSupport         Role = "support"
	RoleFinance         Role = "finance"
	RoleSuperadmin      Role = "superadmin"
)

// Account statuses from the users table.
const (
	StatusActive  = "active"
	StatusBlocked = "blocked"
)

var (
	validLocales    = map[string]bool{"uz": true, "ru": true, "en": true}
	validCurrencies = map[string]bool{"UZS": true}
)

// User is an account row. Credentials stay inside the domain and are never serialized.
type User struct {
	ID              uuid.UUID
	Email           string
	PasswordHash    *string
	GoogleSub       *string
	Name            string
	Phone           *string
	AvatarURL       *string
	Locale          string
	Currency        string
	EmailVerifiedAt *time.Time
	Status          string
}

// Account is a user together with the roles granted to it.
type Account struct {
	User
	Roles []Role
}

// HasRole reports whether the account holds at least one of the roles.
func (a Account) HasRole(roles ...Role) bool {
	for _, want := range roles {
		for _, have := range a.Roles {
			if have == want {
				return true
			}
		}
	}
	return false
}

// NewUser is the input for creating an account.
type NewUser struct {
	Email         string
	PasswordHash  *string
	GoogleSub     *string
	Name          string
	Locale        string
	Currency      string
	EmailVerified bool
}

// ProfileUpdate holds the mutable profile fields. Nil means unchanged.
type ProfileUpdate struct {
	Name     *string
	Phone    *string
	Locale   *string
	Currency *string
}

// RefreshToken is a stored, hashed refresh token. Rotation keeps FamilyID stable.
type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	FamilyID  uuid.UUID
	ExpiresAt time.Time
	RevokedAt *time.Time
	UserAgent string
	IP        string
}

// RequestMeta carries client details recorded with sessions and used for rate limiting.
type RequestMeta struct {
	IP        string
	UserAgent string
}

// Session is the result of a successful sign-in, registration or refresh.
type Session struct {
	Account      Account
	AccessToken  string
	RefreshToken string
}

// GoogleIdentity is the verified subject of a Google ID token.
type GoogleIdentity struct {
	Subject       string
	Email         string
	EmailVerified bool
	Name          string
}

// LocaleValid reports whether a locale is supported.
func LocaleValid(locale string) bool { return validLocales[locale] }

// CurrencyValid reports whether a currency is supported.
func CurrencyValid(currency string) bool { return validCurrencies[currency] }
