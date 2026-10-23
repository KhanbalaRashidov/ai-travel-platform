package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisKV is a Redis-backed KV implementation.
type RedisKV struct {
	c *redis.Client
}

// NewRedisKV wraps a redis client as a KV.
func NewRedisKV(c *redis.Client) *RedisKV { return &RedisKV{c: c} }

// Get returns the value or ErrMiss when the key does not exist.
func (r *RedisKV) Get(ctx context.Context, key string) (string, error) {
	v, err := r.c.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrMiss
	}
	return v, err
}

// Set stores a value with an optional TTL (ttl <= 0 => no expiry).
func (r *RedisKV) Set(ctx context.Context, key, val string, ttl time.Duration) error {
	return r.c.Set(ctx, key, val, ttl).Err()
}

// Del removes a key.
func (r *RedisKV) Del(ctx context.Context, key string) error {
	return r.c.Del(ctx, key).Err()
}

var _ KV = (*RedisKV)(nil)
