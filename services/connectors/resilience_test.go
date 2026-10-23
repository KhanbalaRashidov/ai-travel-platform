package connectors

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetry_SucceedsOnThirdAttempt(t *testing.T) {
	calls := 0
	err := Retry(context.Background(), 3, time.Millisecond, func() error {
		calls++
		if calls < 3 {
			return errors.New("transient")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
}

func TestRetry_ExhaustsAndReturnsLastError(t *testing.T) {
	calls := 0
	want := errors.New("boom")
	err := Retry(context.Background(), 2, time.Millisecond, func() error {
		calls++
		return want
	})
	if !errors.Is(err, want) {
		t.Fatalf("expected %v, got %v", want, err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}

func TestCircuitBreaker_OpensAndRecovers(t *testing.T) {
	now := time.Unix(1000, 0)
	b := NewCircuitBreaker(2, time.Minute)
	b.now = func() time.Time { return now }

	if !b.Allow() {
		t.Fatal("should allow initially")
	}
	b.Failure()
	b.Failure() // reaches threshold -> opens
	if b.Allow() {
		t.Fatal("should be open after max failures")
	}
	now = now.Add(time.Minute) // cooldown elapsed -> half-open
	if !b.Allow() {
		t.Fatal("should allow after cooldown")
	}
	b.Success()
	if !b.Allow() {
		t.Fatal("should allow after success resets failures")
	}
}
