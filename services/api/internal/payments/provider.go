// Package payments defines the provider port used by checkout. Domain code depends only on
// Provider; concrete providers (mock now, Payme, Click and Uzum later) live beside it.
package payments

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
)

// Outcome is the final result a provider reports for one payment attempt.
type Outcome string

const (
	OutcomeSucceeded Outcome = "succeeded"
	OutcomeFailed    Outcome = "failed"
)

// ErrInvalidSignature means a webhook did not carry a valid signature. Nothing is processed.
var ErrInvalidSignature = errors.New("payments: invalid webhook signature")

// ErrMalformedWebhook means a correctly signed webhook body could not be understood.
var ErrMalformedWebhook = errors.New("payments: malformed webhook")

// PaymentRequest describes one payment attempt to create at a provider. It is deliberately not the
// commerce Order type so that providers never import the domain.
type PaymentRequest struct {
	PaymentID   uuid.UUID
	OrderNumber string
	Amount      int64
	Currency    string
}

// Session is what the provider returns after a payment is created. RedirectURL is where the buyer
// continues the payment (a hosted page, or the mock checkout page).
type Session struct {
	ProviderPaymentID string
	RedirectURL       string
	Instructions      string
}

// Event is a verified provider notification about one payment attempt.
type Event struct {
	ProviderPaymentID string
	Outcome           Outcome
}

// Reference identifies a payment at its provider for refunds.
type Reference struct {
	PaymentID         uuid.UUID
	ProviderPaymentID string
}

// Provider is the payment port. Adding a provider means implementing this interface and wiring it
// in configuration; nothing else changes.
type Provider interface {
	Name() string
	CreatePayment(ctx context.Context, request PaymentRequest) (Session, error)
	// ParseWebhook verifies the signature of the request and decodes the event. It returns
	// ErrInvalidSignature or ErrMalformedWebhook on failure.
	ParseWebhook(ctx context.Context, r *http.Request) (Event, error)
	Refund(ctx context.Context, payment Reference, amount int64) error
}
