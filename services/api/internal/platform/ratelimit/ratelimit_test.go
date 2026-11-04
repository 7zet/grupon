package ratelimit

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func TestRedisLimiterCountsWithinWindow(t *testing.T) {
	url := os.Getenv("TEST_REDIS_URL")
	if url == "" {
		t.Skip("TEST_REDIS_URL is not set")
	}
	options, err := redis.ParseURL(url)
	if err != nil {
		t.Fatal(err)
	}
	client := redis.NewClient(options)
	t.Cleanup(func() { _ = client.Close() })

	ctx := context.Background()
	limiter := NewRedis(client, 2, time.Minute)
	key := "test:" + uuid.NewString()
	t.Cleanup(func() { client.Del(ctx, keyPrefix+key) })

	for i := 1; i <= 2; i++ {
		ok, err := limiter.Allow(ctx, key)
		if err != nil || !ok {
			t.Fatalf("attempt %d = %v, %v; want allowed", i, ok, err)
		}
	}
	ok, err := limiter.Allow(ctx, key)
	if err != nil || ok {
		t.Fatalf("attempt 3 = %v, %v; want denied", ok, err)
	}
	if ttl := client.TTL(ctx, keyPrefix+key).Val(); ttl <= 0 || ttl > time.Minute {
		t.Fatalf("ttl = %v, want a window expiry", ttl)
	}
}

func TestDisabledLimiterAllowsEverything(t *testing.T) {
	ok, err := Disabled{}.Allow(context.Background(), "any")
	if err != nil || !ok {
		t.Fatalf("Disabled.Allow() = %v, %v", ok, err)
	}
}
