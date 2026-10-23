// Package dataflow is the self-growing "data flywheel": every lookup checks a
// hot cache (Redis), then a warm durable store (the knowledge base / DB), then a
// cold source (a connector that may hit the network). A cold result is written
// back to BOTH the warm store and the hot cache, so the next identical lookup is
// served without a network call and the knowledge base grows with use.
//
// It depends only on small local interfaces (KV, RouteStore, FxStore) plus the
// connector DTOs, so it never imports the knowledge or gateway packages — the
// concrete PgStore/MemStore satisfy RouteStore/FxStore structurally.
package dataflow

import (
	"context"
	"encoding/json"
	"time"

	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/cache"
	conn "github.com/KhanbalaRashidov/ai-travel-platform/services/connectors"
)

// Tier reports which layer served a result (observability + UI "freshness").
type Tier string

const (
	TierHot  Tier = "hot"  // Redis
	TierWarm Tier = "warm" // durable knowledge base / DB
	TierCold Tier = "cold" // connector (network / estimate)
)

// freshable is satisfied by any DTO embedding connectors.Provenance.
type freshable interface {
	Fresh(now time.Time) bool
}

// RouteStore is the warm layer for transport quotes (knowledge.Store satisfies it).
type RouteStore interface {
	GetRoute(ctx context.Context, tenantID, from, to string, mode conn.TransportMode) (conn.TransportQuote, bool, error)
	PutRoute(ctx context.Context, tenantID string, q conn.TransportQuote) error
}

// FxStore is the warm layer for FX rates (knowledge.Store satisfies it).
type FxStore interface {
	GetFx(ctx context.Context, tenantID, base, quote string) (conn.FxRate, bool, error)
	PutFx(ctx context.Context, tenantID string, r conn.FxRate) error
}

// resolve runs the hot -> warm -> cold read-through with write-back. It is the
// single place the flywheel logic lives; the typed flywheels below just supply
// the per-domain closures.
func resolve[T freshable](
	ctx context.Context,
	kv cache.KV,
	key string,
	hotTTL time.Duration,
	now time.Time,
	warmGet func() (T, bool, error),
	warmPut func(T) error,
	cold func() (T, error),
) (T, Tier, error) {
	var zero T

	// 1) Hot cache (Redis). A decode failure or stale value is treated as a miss.
	if raw, err := kv.Get(ctx, key); err == nil {
		var v T
		if json.Unmarshal([]byte(raw), &v) == nil && v.Fresh(now) {
			return v, TierHot, nil
		}
	}

	// 2) Warm durable store. ok=false already means missing OR stale.
	if v, ok, err := warmGet(); err != nil {
		return zero, "", err
	} else if ok && v.Fresh(now) {
		setHot(ctx, kv, key, v, hotTTL)
		return v, TierWarm, nil
	}

	// 3) Cold source; write back to warm + hot so it is reused next time.
	v, err := cold()
	if err != nil {
		return zero, "", err
	}
	if warmPut != nil {
		_ = warmPut(v) // durable write is best-effort; a failure must not drop the result
	}
	setHot(ctx, kv, key, v, hotTTL)
	return v, TierCold, nil
}

// setHot writes a value to the hot cache (best-effort; cache failures are non-fatal).
func setHot[T any](ctx context.Context, kv cache.KV, key string, v T, ttl time.Duration) {
	if b, err := json.Marshal(v); err == nil {
		_ = kv.Set(ctx, key, string(b), ttl)
	}
}

// RouteFlywheel resolves transport quotes through hot -> warm -> cold.
type RouteFlywheel struct {
	kv     cache.KV
	store  RouteStore
	source conn.TransportSource
	hotTTL time.Duration
	now    func() time.Time
}

// NewRouteFlywheel wires the transport flywheel. hotTTL <=0 defaults to 10m.
func NewRouteFlywheel(kv cache.KV, store RouteStore, source conn.TransportSource, hotTTL time.Duration) *RouteFlywheel {
	if hotTTL <= 0 {
		hotTTL = 10 * time.Minute
	}
	return &RouteFlywheel{kv: kv, store: store, source: source, hotTTL: hotTTL, now: time.Now}
}

// Route returns a transport quote and the tier that served it.
func (f *RouteFlywheel) Route(ctx context.Context, tenantID, from, to string, mode conn.TransportMode) (conn.TransportQuote, Tier, error) {
	key := "route|" + tenantID + "|" + from + "|" + to + "|" + string(mode)
	return resolve[conn.TransportQuote](
		ctx, f.kv, key, f.hotTTL, f.now().UTC(),
		func() (conn.TransportQuote, bool, error) {
			return f.store.GetRoute(ctx, tenantID, from, to, mode)
		},
		func(q conn.TransportQuote) error { return f.store.PutRoute(ctx, tenantID, q) },
		func() (conn.TransportQuote, error) { return f.source.Transport(ctx, from, to, mode) },
	)
}

// FxFlywheel resolves FX rates through hot -> warm -> cold.
type FxFlywheel struct {
	kv     cache.KV
	store  FxStore
	source conn.FxSource
	hotTTL time.Duration
	now    func() time.Time
}

// NewFxFlywheel wires the FX flywheel. hotTTL <=0 defaults to 1h.
func NewFxFlywheel(kv cache.KV, store FxStore, source conn.FxSource, hotTTL time.Duration) *FxFlywheel {
	if hotTTL <= 0 {
		hotTTL = time.Hour
	}
	return &FxFlywheel{kv: kv, store: store, source: source, hotTTL: hotTTL, now: time.Now}
}

// Rate returns an FX rate and the tier that served it.
func (f *FxFlywheel) Rate(ctx context.Context, tenantID, base, quote string) (conn.FxRate, Tier, error) {
	key := "fx|" + tenantID + "|" + base + "|" + quote
	return resolve[conn.FxRate](
		ctx, f.kv, key, f.hotTTL, f.now().UTC(),
		func() (conn.FxRate, bool, error) { return f.store.GetFx(ctx, tenantID, base, quote) },
		func(r conn.FxRate) error { return f.store.PutFx(ctx, tenantID, r) },
		func() (conn.FxRate, error) { return f.source.Rate(ctx, base, quote) },
	)
}
