package cache

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMemKV_SetGetDel(t *testing.T) {
	ctx := context.Background()
	c := NewMemKV()

	if _, err := c.Get(ctx, "nope"); !errors.Is(err, ErrMiss) {
		t.Fatalf("expected miss, got %v", err)
	}
	if err := c.Set(ctx, "k", "v", 0); err != nil {
		t.Fatal(err)
	}
	if v, err := c.Get(ctx, "k"); err != nil || v != "v" {
		t.Fatalf("expected v, got %q (%v)", v, err)
	}
	if err := c.Del(ctx, "k"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Get(ctx, "k"); !errors.Is(err, ErrMiss) {
		t.Fatalf("expected miss after del, got %v", err)
	}
}

func TestMemKV_TTLExpiry(t *testing.T) {
	ctx := context.Background()
	now := time.Unix(1000, 0)
	c := NewMemKV()
	c.now = func() time.Time { return now }

	if err := c.Set(ctx, "k", "v", time.Minute); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Get(ctx, "k"); err != nil {
		t.Fatalf("expected hit before expiry, got %v", err)
	}
	now = now.Add(time.Minute) // at expiry boundary -> expired
	if _, err := c.Get(ctx, "k"); !errors.Is(err, ErrMiss) {
		t.Fatalf("expected miss after expiry, got %v", err)
	}
}
