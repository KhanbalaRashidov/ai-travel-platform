package cache

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrMiss is returned by KV.Get when the key is absent or expired.
var ErrMiss = errors.New("cache miss")

// KV is a minimal string key/value cache with TTL. Both a Redis-backed and an
// in-memory implementation satisfy it, so services never hard-depend on Redis
// (handy for Docker-free tests and dev).
type KV interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, val string, ttl time.Duration) error
	Del(ctx context.Context, key string) error
}

// MemKV is an in-memory KV with per-key expiry (tests / dev without Redis).
type MemKV struct {
	mu  sync.RWMutex
	m   map[string]memItem
	now func() time.Time
}

type memItem struct {
	val     string
	expires time.Time // zero => no expiry
}

// NewMemKV builds an empty in-memory KV.
func NewMemKV() *MemKV {
	return &MemKV{m: make(map[string]memItem), now: time.Now}
}

// Get returns the value or ErrMiss if absent/expired.
func (c *MemKV) Get(_ context.Context, key string) (string, error) {
	c.mu.RLock()
	it, ok := c.m[key]
	c.mu.RUnlock()
	if !ok {
		return "", ErrMiss
	}
	if !it.expires.IsZero() && !c.now().Before(it.expires) {
		c.mu.Lock()
		delete(c.m, key)
		c.mu.Unlock()
		return "", ErrMiss
	}
	return it.val, nil
}

// Set stores a value with an optional TTL (ttl <= 0 => no expiry).
func (c *MemKV) Set(_ context.Context, key, val string, ttl time.Duration) error {
	var exp time.Time
	if ttl > 0 {
		exp = c.now().Add(ttl)
	}
	c.mu.Lock()
	c.m[key] = memItem{val: val, expires: exp}
	c.mu.Unlock()
	return nil
}

// Del removes a key.
func (c *MemKV) Del(_ context.Context, key string) error {
	c.mu.Lock()
	delete(c.m, key)
	c.mu.Unlock()
	return nil
}

var _ KV = (*MemKV)(nil)
