package dataflow

import (
	"context"
	"testing"
	"time"

	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/cache"
	conn "github.com/KhanbalaRashidov/ai-travel-platform/services/connectors"
)

// --- fakes -----------------------------------------------------------------

type fakeSource struct {
	calls int
	q     conn.TransportQuote
	err   error
}

func (f *fakeSource) Name() string { return "fake" }
func (f *fakeSource) Transport(_ context.Context, _, _ string, _ conn.TransportMode) (conn.TransportQuote, error) {
	f.calls++
	if f.err != nil {
		return conn.TransportQuote{}, f.err
	}
	return f.q, nil
}

type fakeRouteStore struct {
	m        map[string]conn.TransportQuote
	getCalls int
	putCalls int
}

func newFakeRouteStore() *fakeRouteStore {
	return &fakeRouteStore{m: map[string]conn.TransportQuote{}}
}
func (s *fakeRouteStore) GetRoute(_ context.Context, tenantID, from, to string, mode conn.TransportMode) (conn.TransportQuote, bool, error) {
	s.getCalls++
	q, ok := s.m[tenantID+from+to+string(mode)]
	if !ok || !q.Fresh(time.Now()) {
		return conn.TransportQuote{}, false, nil
	}
	return q, true, nil
}
func (s *fakeRouteStore) PutRoute(_ context.Context, tenantID string, q conn.TransportQuote) error {
	s.putCalls++
	s.m[tenantID+q.From+q.To+string(q.Mode)] = q
	return nil
}

func quote(expiresIn time.Duration) conn.TransportQuote {
	return conn.TransportQuote{
		From: "A", To: "B", Mode: conn.ModeFlight,
		Price: conn.Money{Amount: 100, Currency: "EUR"},
		Provenance: conn.Provenance{
			Source:    "fake",
			FetchedAt: time.Now().UTC(),
			ExpiresAt: time.Now().UTC().Add(expiresIn),
		},
	}
}

// --- tests -----------------------------------------------------------------

func TestRouteFlywheel_ColdThenHot(t *testing.T) {
	kv := cache.NewMemKV()
	store := newFakeRouteStore()
	src := &fakeSource{q: quote(time.Hour)}
	fw := NewRouteFlywheel(kv, store, src, time.Minute)
	ctx := context.Background()

	q1, tier, err := fw.Route(ctx, "t1", "A", "B", conn.ModeFlight)
	if err != nil {
		t.Fatalf("cold route: %v", err)
	}
	if tier != TierCold {
		t.Fatalf("first lookup should be cold, got %s", tier)
	}
	if q1.Price.Amount != 100 {
		t.Fatalf("unexpected price %v", q1.Price.Amount)
	}
	if store.putCalls != 1 {
		t.Fatalf("cold result should be written to warm store once, got %d", store.putCalls)
	}

	// Second identical lookup is served hot; the source is not called again.
	_, tier2, err := fw.Route(ctx, "t1", "A", "B", conn.ModeFlight)
	if err != nil {
		t.Fatalf("hot route: %v", err)
	}
	if tier2 != TierHot {
		t.Fatalf("second lookup should be hot, got %s", tier2)
	}
	if src.calls != 1 {
		t.Fatalf("source must be called exactly once, got %d", src.calls)
	}
}

func TestRouteFlywheel_WarmHitPopulatesHot(t *testing.T) {
	kv := cache.NewMemKV()
	store := newFakeRouteStore()
	_ = store.PutRoute(context.Background(), "t1", quote(time.Hour)) // warm pre-seeded
	src := &fakeSource{err: conn.ErrUnavailable}                     // must not be needed
	fw := NewRouteFlywheel(kv, store, src, time.Minute)
	ctx := context.Background()

	_, tier, err := fw.Route(ctx, "t1", "A", "B", conn.ModeFlight)
	if err != nil {
		t.Fatalf("warm route: %v", err)
	}
	if tier != TierWarm {
		t.Fatalf("expected warm, got %s", tier)
	}
	if src.calls != 0 {
		t.Fatalf("source must not be called on a warm hit, got %d", src.calls)
	}
	// Now hot is populated.
	_, tier2, _ := fw.Route(ctx, "t1", "A", "B", conn.ModeFlight)
	if tier2 != TierHot {
		t.Fatalf("expected hot after warm write-back, got %s", tier2)
	}
}

func TestRouteFlywheel_StaleHotFallsThrough(t *testing.T) {
	kv := cache.NewMemKV()
	store := newFakeRouteStore()
	src := &fakeSource{q: quote(time.Hour)}
	fw := NewRouteFlywheel(kv, store, src, time.Hour)
	ctx := context.Background()

	// Seed hot with an already-expired provenance (KV TTL long, but Fresh=false).
	setHot(ctx, kv, "route|t1|A|B|flight", quote(-time.Minute), time.Hour)

	_, tier, err := fw.Route(ctx, "t1", "A", "B", conn.ModeFlight)
	if err != nil {
		t.Fatalf("route: %v", err)
	}
	if tier == TierHot {
		t.Fatal("a stale hot entry must not be served as hot")
	}
	if tier != TierCold {
		t.Fatalf("expected cold (empty warm), got %s", tier)
	}
}

func TestRouteFlywheel_SourceErrorPropagates(t *testing.T) {
	kv := cache.NewMemKV()
	store := newFakeRouteStore()
	src := &fakeSource{err: conn.ErrUnavailable}
	fw := NewRouteFlywheel(kv, store, src, time.Minute)

	_, _, err := fw.Route(context.Background(), "t1", "A", "B", conn.ModeFlight)
	if err == nil {
		t.Fatal("expected error when every tier misses and the source fails")
	}
}
