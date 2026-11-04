// Package outbox delivers the events written next to each ledger entry (ledger.posted and friends).
// Delivery is at-least-once: a batch is marked published only after every event in it was handed to
// the publisher, so consumers must ignore events they have already seen.
package outbox

import (
	"context"
	"fmt"
)

// Publisher hands one event to its transport (a broker, a webhook, or a log for now).
type Publisher func(ctx context.Context, topic string, payload []byte) error

// Store claims a batch of unpublished events, publishes them and marks them, in one transaction.
type Store interface {
	PublishPending(ctx context.Context, batch int, publish Publisher) (int, error)
}

// Relay moves pending events from the outbox table to a publisher.
type Relay struct {
	store   Store
	publish Publisher
}

// NewRelay binds the outbox store to a publisher.
func NewRelay(store Store, publish Publisher) *Relay {
	return &Relay{store: store, publish: publish}
}

// Run publishes up to batch events and returns how many were delivered.
func (r *Relay) Run(ctx context.Context, batch int) (int, error) {
	if batch < 1 {
		return 0, fmt.Errorf("outbox: batch must be positive")
	}
	return r.store.PublishPending(ctx, batch, r.publish)
}
