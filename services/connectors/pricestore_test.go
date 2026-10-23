package connectors

import (
	"context"
	"testing"
	"time"
)

func TestMemPriceStore_FxFreshness(t *testing.T) {
	now := time.Unix(10_000, 0)
	s := NewMemPriceStore()
	s.now = func() time.Time { return now }
	ctx := context.Background()

	fresh := FxRate{Base: "USD", Quote: "EUR", Rate: 0.92,
		Provenance: Provenance{Source: "x", FetchedAt: now, ExpiresAt: now.Add(time.Hour)}}
	if err := s.PutFx(ctx, "t1", fresh); err != nil {
		t.Fatal(err)
	}
	if got, ok, _ := s.GetFx(ctx, "t1", "USD", "EUR"); !ok || got.Rate != 0.92 {
		t.Fatalf("expected fresh hit, got ok=%v rate=%v", ok, got.Rate)
	}
	// tenant isolation
	if _, ok, _ := s.GetFx(ctx, "t2", "USD", "EUR"); ok {
		t.Fatal("tenant t2 must not see t1 data")
	}
	// stale entry is a miss
	now = now.Add(2 * time.Hour)
	if _, ok, _ := s.GetFx(ctx, "t1", "USD", "EUR"); ok {
		t.Fatal("expected stale miss")
	}
}

func TestMemPriceStore_Transport(t *testing.T) {
	now := time.Unix(10_000, 0)
	s := NewMemPriceStore()
	s.now = func() time.Time { return now }
	ctx := context.Background()

	q := TransportQuote{From: "BAK", To: "IST", Mode: ModeFlight,
		Price: Money{Amount: 120, Currency: "EUR"}, Duration: 2 * time.Hour,
		Provenance: Provenance{Source: "x", FetchedAt: now, ExpiresAt: now.Add(time.Hour)}}
	if err := s.PutTransport(ctx, "t1", q); err != nil {
		t.Fatal(err)
	}
	got, ok, _ := s.GetTransport(ctx, "t1", "BAK", "IST", ModeFlight)
	if !ok || got.Price.Amount != 120 {
		t.Fatalf("expected hit, got ok=%v amount=%v", ok, got.Price.Amount)
	}
}
