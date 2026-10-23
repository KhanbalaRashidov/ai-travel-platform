package pricefetch

import (
	"context"
	"sync"
	"time"
)

// MemWarm is an in-memory Warm tier (tests / dev without a database).
type MemWarm struct {
	mu  sync.RWMutex
	m   map[string]warmItem
	now func() time.Time
}

type warmItem struct {
	cand    Candidate
	expires time.Time
}

// NewMemWarm builds an empty in-memory warm cache.
func NewMemWarm() *MemWarm { return &MemWarm{m: make(map[string]warmItem), now: time.Now} }

func warmKey(tenantID, key string) string { return tenantID + "\x00" + key }

// GetFare returns a fresh cached fare if present.
func (w *MemWarm) GetFare(_ context.Context, tenantID, key string) (Candidate, bool) {
	w.mu.RLock()
	it, ok := w.m[warmKey(tenantID, key)]
	w.mu.RUnlock()
	if !ok || (!it.expires.IsZero() && !w.now().Before(it.expires)) {
		return Candidate{}, false
	}
	return it.cand, true
}

// PutFare stores a fare with a TTL.
func (w *MemWarm) PutFare(_ context.Context, tenantID, key string, c Candidate, ttl time.Duration) {
	var exp time.Time
	if ttl > 0 {
		exp = w.now().Add(ttl)
	}
	w.mu.Lock()
	w.m[warmKey(tenantID, key)] = warmItem{cand: c, expires: exp}
	w.mu.Unlock()
}

var _ Warm = (*MemWarm)(nil)
