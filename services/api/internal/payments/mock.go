package payments

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
)

// MockSignatureHeader carries the hex HMAC-SHA256 of the raw webhook body.
const MockSignatureHeader = "X-Webhook-Signature"

// maxMockWebhookBytes bounds the body read from a webhook request.
const maxMockWebhookBytes = 16 << 10

// MockPaymentPath is the buyer-facing page of the mock provider. The web app renders it.
const MockPaymentPath = "/checkout/mock/"

// Mock is the test payment provider. It creates payments that the buyer completes with two
// buttons, and it signs webhooks exactly like a real provider would, so the verification path is
// the same as in production.
type Mock struct {
	secret []byte
}

// NewMock returns a mock provider signing webhooks with secret.
func NewMock(secret string) (*Mock, error) {
	if len(secret) < 32 {
		return nil, errors.New("payments: mock webhook secret must be at least 32 bytes")
	}
	return &Mock{secret: []byte(secret)}, nil
}

var _ Provider = (*Mock)(nil)

// mockWebhook is the body of a mock webhook.
type mockWebhook struct {
	ProviderPaymentID string  `json:"provider_payment_id"`
	Outcome           Outcome `json:"outcome"`
}

// Name identifies the provider in routes and in the payments table.
func (m *Mock) Name() string { return "mock" }

// CreatePayment records a pending mock payment whose provider id is the payment id.
func (m *Mock) CreatePayment(_ context.Context, request PaymentRequest) (Session, error) {
	if request.PaymentID == uuid.Nil {
		return Session{}, errors.New("payments: mock payment needs a payment id")
	}
	id := request.PaymentID.String()
	return Session{ProviderPaymentID: id, RedirectURL: MockPaymentPath + id}, nil
}

// ParseWebhook verifies the HMAC of the body and decodes the event.
func (m *Mock) ParseWebhook(_ context.Context, r *http.Request) (Event, error) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxMockWebhookBytes+1))
	if err != nil {
		return Event{}, fmt.Errorf("read mock webhook: %w", err)
	}
	if len(body) > maxMockWebhookBytes {
		return Event{}, ErrMalformedWebhook
	}
	if !m.validSignature(body, r.Header.Get(MockSignatureHeader)) {
		return Event{}, ErrInvalidSignature
	}
	var payload mockWebhook
	if err := json.Unmarshal(body, &payload); err != nil {
		return Event{}, ErrMalformedWebhook
	}
	if payload.ProviderPaymentID == "" || (payload.Outcome != OutcomeSucceeded && payload.Outcome != OutcomeFailed) {
		return Event{}, ErrMalformedWebhook
	}
	return Event{ProviderPaymentID: payload.ProviderPaymentID, Outcome: payload.Outcome}, nil
}

// Refund is a no-op for the mock: no money moves, and the payment row is updated by the caller.
func (m *Mock) Refund(_ context.Context, _ Reference, _ int64) error { return nil }

// SignedWebhook builds the body and signature the mock sends for an outcome. The caller delivers
// them to the same webhook handler a real provider would call.
func (m *Mock) SignedWebhook(providerPaymentID string, outcome Outcome) ([]byte, string, error) {
	body, err := json.Marshal(mockWebhook{ProviderPaymentID: providerPaymentID, Outcome: outcome})
	if err != nil {
		return nil, "", fmt.Errorf("encode mock webhook: %w", err)
	}
	return body, m.sign(body), nil
}

func (m *Mock) sign(body []byte) string {
	mac := hmac.New(sha256.New, m.secret)
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func (m *Mock) validSignature(body []byte, signature string) bool {
	expected := m.sign(body)
	return hmac.Equal([]byte(expected), bytes.ToLower([]byte(signature)))
}
