package connectors

import (
	"context"
	"sync"
	"time"
)

// PriceStore persists and retrieves quotes — the "price store". Lookups return
// ok=false for missing OR stale entries, so callers transparently re-fetch.
// A DB-backed implementation lands in E6 (knowledge base); the in-memory
// implementation below is used for tests and Docker-free dev.
type PriceStore interface {
	PutTransport(ctx context.Context, tenantID string, q TransportQuote) error
	GetTransport(ctx context.Context, tenantID, from, to string, mode TransportMode) (TransportQuote, bool, error)
	PutFx(ctx context.Context, tenantID string, r FxRate) error
	GetFx(ctx context.Context, tenantID, base, quote string) (FxRate, bool, error)
}

// MemPriceStore is an in-memory PriceStore (tests / dev without a database).
type MemPriceStore struct {
	mu        sync.RWMutex
	transport map[string]TransportQuote
	fx        map[string]FxRate
	now       func() time.Time
}

// NewMemPriceStore constructs an empty in-memory price store.
func NewMemPriceStore() *MemPriceStore {
	return &MemPriceStore{
		transport: make(map[string]TransportQuote),
		fx:        make(map[string]FxRate),
		now:       time.Now,
	}
}

func transportKey(tenantID, from, to string, mode TransportMode) string {
	return tenantID + "|" + from + "|" + to + "|" + string(mode)
}

func fxKey(tenantID, base, quote string) string {
	return tenantID + "|" + base + "|" + quote
}

// PutTransport stores a transport quote.
func (s *MemPriceStore) PutTransport(_ context.Context, tenantID string, q TransportQuote) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.transport[transportKey(tenantID, q.From, q.To, q.Mode)] = q
	return nil
}

// GetTransport returns a fresh transport quote if present.
func (s *MemPriceStore) GetTransport(_ context.Context, tenantID, from, to string, mode TransportMode) (TransportQuote, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	q, ok := s.transport[transportKey(tenantID, from, to, mode)]
	if !ok || !q.Fresh(s.now()) {
		return TransportQuote{}, false, nil
	}
	return q, true, nil
}

// PutFx stores an FX rate.
func (s *MemPriceStore) PutFx(_ context.Context, tenantID string, r FxRate) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fx[fxKey(tenantID, r.Base, r.Quote)] = r
	return nil
}

// GetFx returns a fresh FX rate if present.
func (s *MemPriceStore) GetFx(_ context.Context, tenantID, base, quote string) (FxRate, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.fx[fxKey(tenantID, base, quote)]
	if !ok || !r.Fresh(s.now()) {
		return FxRate{}, false, nil
	}
	return r, true, nil
}

var _ PriceStore = (*MemPriceStore)(nil)
