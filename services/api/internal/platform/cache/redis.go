// Package cache provides the Redis read-through cache used by public catalog reads.
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	namespace      = "catalog"
	versionKey     = namespace + ":version"
	initialVersion = "0"
)

// Redis caches JSON values under a namespace version. Invalidate bumps the version, so every
// previously written entry becomes unreachable without scanning or deleting keys.
type Redis struct {
	client *redis.Client
}

// NewRedis wraps a Redis client. The client is owned by the caller.
func NewRedis(client *redis.Client) *Redis {
	return &Redis{client: client}
}

// Get decodes a cached value into dst. A missing entry is reported as a miss, not an error.
func (c *Redis) Get(ctx context.Context, key string, dst any) (bool, error) {
	version, err := c.version(ctx)
	if err != nil {
		return false, err
	}
	raw, err := c.client.Get(ctx, entryKey(version, key)).Bytes()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("get cache entry: %w", err)
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		return false, fmt.Errorf("decode cache entry: %w", err)
	}
	return true, nil
}

// Set stores a JSON value for ttl under the current namespace version.
func (c *Redis) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	version, err := c.version(ctx)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode cache entry: %w", err)
	}
	if err := c.client.Set(ctx, entryKey(version, key), raw, ttl).Err(); err != nil {
		return fmt.Errorf("set cache entry: %w", err)
	}
	return nil
}

// Invalidate makes all catalog cache entries stale.
func (c *Redis) Invalidate(ctx context.Context) error {
	if err := c.client.Incr(ctx, versionKey).Err(); err != nil {
		return fmt.Errorf("invalidate catalog cache: %w", err)
	}
	return nil
}

// Ping verifies that Redis accepts connections.
func (c *Redis) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

func (c *Redis) version(ctx context.Context) (string, error) {
	version, err := c.client.Get(ctx, versionKey).Result()
	if errors.Is(err, redis.Nil) {
		return initialVersion, nil
	}
	if err != nil {
		return "", fmt.Errorf("read cache version: %w", err)
	}
	return version, nil
}

func entryKey(version, key string) string {
	return fmt.Sprintf("%s:v%s:%s", namespace, version, key)
}
