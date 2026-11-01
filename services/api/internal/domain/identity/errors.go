// Package identity owns accounts, sessions, roles and account recovery rules.
package identity

import "errors"

// Sentinel errors. The HTTP layer maps them to stable problem codes.
var (
	ErrNotFound             = errors.New("identity: not found")
	ErrEmailTaken           = errors.New("identity: email already registered")
	ErrInvalidCredentials   = errors.New("identity: invalid credentials")
	ErrCurrentPasswordWrong = errors.New("identity: current password incorrect")
	ErrAccountBlocked       = errors.New("identity: account blocked")
	ErrInvalidToken         = errors.New("identity: invalid or expired token")
	ErrRefreshTokenInvalid  = errors.New("identity: refresh token invalid")
	ErrRefreshTokenReused   = errors.New("identity: refresh token reused")
	ErrUnauthenticated      = errors.New("identity: unauthenticated")
	ErrForbidden            = errors.New("identity: forbidden")
	ErrRateLimited          = errors.New("identity: rate limited")
)

// ValidationError reports one invalid input field with a stable code.
type ValidationError struct {
	Field string
	Code  string
}

func (e *ValidationError) Error() string {
	return "identity: invalid " + e.Field + ": " + e.Code
}
