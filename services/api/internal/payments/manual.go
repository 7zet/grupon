package payments

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
)

// Manual represents the pilot bank-transfer mode. It never accepts public webhooks; an authorized
// staff confirmation is converted to the same verified Event used by webhook settlement.
type Manual struct{ Instructions string }

func (m *Manual) Name() string { return "manual" }
func (m *Manual) CreatePayment(_ context.Context, request PaymentRequest) (Session, error) {
	if request.PaymentID == uuid.Nil {
		return Session{}, errors.New("payments: manual payment needs an id")
	}
	return Session{ProviderPaymentID: request.PaymentID.String(), Instructions: m.Instructions}, nil
}
func (*Manual) ParseWebhook(context.Context, *http.Request) (Event, error) {
	return Event{}, ErrInvalidSignature
}
func (*Manual) Refund(context.Context, Reference, int64) error {
	return errors.New("payments: manual refund required")
}
