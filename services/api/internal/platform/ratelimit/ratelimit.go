// Package ratelimit provides fixed-window attempt counters backed by Redis.
package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const keyPrefix = "ratelimit:"

// Redis counts attempts per key in fixed windows. The first attempt sets the window expiry.
type Redis struct {
	client *redis.Client
	limit  int64
	window time.Duration
}

// NewRedis returns a limiter that allows limit attempts per window for each key.
func NewRedis(client *redis.Client, limit int, window time.Duration) *Redis {
	return &Redis{client: client, limit: int64(limit), window: window}
}

// Allow records one attempt and reports whether it is within the limit.
func (r *Redis) Allow(ctx context.Context, key string) (bool, error) {
	fullKey := keyPrefix + key
	pipe := r.client.TxPipeline()
	incr := pipe.Incr(ctx, fullKey)
	pipe.ExpireNX(ctx, fullKey, r.window)
	if _, err := pipe.Exec(ctx); err != nil {
		return false, fmt.Errorf("record rate limit attempt: %w", err)
	}
	return incr.Val() <= r.limit, nil
}

// Disabled allows every attempt. It is used when Redis is not configured.
type Disabled struct{}

// Allow always permits the attempt.
func (Disabled) Allow(context.Context, string) (bool, error) { return true, nil }
