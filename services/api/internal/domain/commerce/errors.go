// Package commerce implements carts, orders, payments, vouchers and the background sweeps that keep
// stock reservations honest. It has no HTTP or SQL knowledge.
package commerce

import (
	"errors"
	"fmt"

	"github.com/grupon/grupon/services/api/internal/payments"
)

var (
	ErrNotFound          = errors.New("commerce: not found")
	ErrOrderNotFound     = errors.New("commerce: order not found")
	ErrPaymentNotFound   = errors.New("commerce: payment not found")
	ErrVoucherNotFound   = errors.New("commerce: voucher not found")
	ErrDealNotFound      = errors.New("commerce: deal not found")
	ErrUserNotFound      = errors.New("commerce: user not found")
	ErrOptionUnavailable = errors.New("commerce: option unavailable")
	ErrDealUnavailable   = errors.New("commerce: deal unavailable")
	// ErrInsufficientStock means the conditional stock reservation matched no rows.
	ErrInsufficientStock       = errors.New("commerce: insufficient stock")
	ErrMaxPerUserExceeded      = errors.New("commerce: max per user exceeded")
	ErrGiftNotAllowed          = errors.New("commerce: deal is not giftable")
	ErrShippingAddressRequired = errors.New("commerce: shipping address required")
	ErrIdempotencyConflict     = errors.New("commerce: idempotency key reused with a different body")
	ErrReservationExpired      = errors.New("commerce: stock reservation expired")
	ErrOrderNotPayable         = errors.New("commerce: order is not awaiting payment")
	ErrPaymentNotPending       = errors.New("commerce: payment is already settled")
	ErrProviderUnavailable     = errors.New("commerce: payment provider unavailable")
	ErrInvalidSignature        = payments.ErrInvalidSignature
	ErrMalformedWebhook        = payments.ErrMalformedWebhook
	ErrInvalidQRToken          = errors.New("commerce: invalid qr token")
	ErrQRExpired               = errors.New("commerce: qr token rotated, fetch a new one")
)

// ValidationError reports one invalid input field with a stable code.
type ValidationError struct {
	Field string
	Code  string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("commerce: %s is invalid (%s)", e.Field, e.Code)
}

func invalid(field, code string) error {
	return &ValidationError{Field: field, Code: code}
}
