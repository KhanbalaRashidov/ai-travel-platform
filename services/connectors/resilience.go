package connectors

import (
	"context"
	"sync"
	"time"
)

// Retry runs fn up to attempts times with exponential backoff (base, 2*base,
// 4*base, ...). It stops early if the context is cancelled and returns the
// last error on exhaustion.
func Retry(ctx context.Context, attempts int, base time.Duration, fn func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		if err = fn(); err == nil {
			return nil
		}
		if i == attempts-1 {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(base * time.Duration(1<<i)):
		}
	}
	return err
}

// CircuitBreaker is a minimal failure-count breaker. After maxFailures
// consecutive failures it opens and rejects calls until cooldown elapses
// (then allows a trial call — half-open).
type CircuitBreaker struct {
	mu          sync.Mutex
	maxFailures int
	cooldown    time.Duration
	failures    int
	openedAt    time.Time
	now         func() time.Time
}

// NewCircuitBreaker builds a breaker.
func NewCircuitBreaker(maxFailures int, cooldown time.Duration) *CircuitBreaker {
	return &CircuitBreaker{maxFailures: maxFailures, cooldown: cooldown, now: time.Now}
}

// Allow reports whether a call may proceed.
func (b *CircuitBreaker) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.failures < b.maxFailures {
		return true
	}
	return b.now().Sub(b.openedAt) >= b.cooldown
}

// Success resets the failure count.
func (b *CircuitBreaker) Success() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures = 0
}

// Failure records a failure and opens the breaker at the threshold.
func (b *CircuitBreaker) Failure() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures++
	if b.failures >= b.maxFailures {
		b.openedAt = b.now()
	}
}
