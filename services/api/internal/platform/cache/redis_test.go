package cache

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// TestRedisRoundTripAndInvalidation needs REDIS_URL. Without it the test is skipped, because the
// cache is an optional dependency of the API.
func TestRedisRoundTripAndInvalidation(t *testing.T) {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		t.Skip("REDIS_URL is not set")
	}
	options, err := redis.ParseURL(url)
	if err != nil {
		t.Fatal(err)
	}
	client := redis.NewClient(options)
	defer client.Close()

	ctx := context.Background()
	c := NewRedis(client)
	if err := c.Ping(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}
	type payload struct {
		Name string `json:"name"`
	}

	key := "test:" + time.Now().Format("150405.000000000")
	if err := c.Set(ctx, key, payload{Name: "spa"}, time.Minute); err != nil {
		t.Fatalf("set: %v", err)
	}
	var got payload
	hit, err := c.Get(ctx, key, &got)
	if err != nil || !hit || got.Name != "spa" {
		t.Fatalf("get = hit:%v name:%q err:%v", hit, got.Name, err)
	}

	if err := c.Invalidate(ctx); err != nil {
		t.Fatalf("invalidate: %v", err)
	}
	hit, err = c.Get(ctx, key, &got)
	if err != nil || hit {
		t.Fatalf("entry survived invalidation: hit=%v err=%v", hit, err)
	}
}
