package knowledge

import (
	"context"
	"testing"
	"time"

	conn "github.com/KhanbalaRashidov/ai-travel-platform/services/connectors"
)

func TestMemStore_RouteFreshnessAndTenant(t *testing.T) {
	now := time.Unix(10_000, 0)
	s := NewMemStore()
	s.now = func() time.Time { return now }
	ctx := context.Background()

	q := conn.TransportQuote{
		From: "BAK", To: "IST", Mode: conn.ModeFlight,
		Price:      conn.Money{Amount: 120, Currency: "EUR"},
		Provenance: conn.Provenance{Source: "x", FetchedAt: now, ExpiresAt: now.Add(time.Hour)},
	}
	if err := s.PutRoute(ctx, "t1", q); err != nil {
		t.Fatal(err)
	}
	if got, ok, _ := s.GetRoute(ctx, "t1", "BAK", "IST", conn.ModeFlight); !ok || got.Price.Amount != 120 {
		t.Fatalf("expected fresh hit, got ok=%v amount=%v", ok, got.Price.Amount)
	}
	if _, ok, _ := s.GetRoute(ctx, "t2", "BAK", "IST", conn.ModeFlight); ok {
		t.Fatal("tenant isolation violated")
	}
	now = now.Add(2 * time.Hour)
	if _, ok, _ := s.GetRoute(ctx, "t1", "BAK", "IST", conn.ModeFlight); ok {
		t.Fatal("stale route must be a miss")
	}
}

func TestMemStore_Destinations(t *testing.T) {
	ctx := context.Background()
	s := NewMemStore()
	_ = s.UpsertDestination(ctx, "t1", Destination{Country: "TR", City: "Istanbul"})
	_ = s.UpsertDestination(ctx, "t1", Destination{Country: "AZ", City: "Baku"})

	ds, _ := s.ListDestinations(ctx, "t1")
	if len(ds) != 2 || ds[0].City != "Baku" { // sorted by country then city: AZ < TR
		t.Fatalf("expected 2 sorted destinations (Baku first), got %+v", ds)
	}
	if other, _ := s.ListDestinations(ctx, "t2"); len(other) != 0 {
		t.Fatalf("tenant t2 should be empty, got %+v", other)
	}
}
