package knowledge

import (
	"context"
	"sort"
	"sync"
	"time"

	conn "github.com/KhanbalaRashidov/ai-travel-platform/services/connectors"
)

// MemStore is an in-memory, tenant-isolated Store for tests and Docker-free dev.
type MemStore struct {
	mu           sync.RWMutex
	routes       map[string]conn.TransportQuote
	fx           map[string]conn.FxRate
	destinations map[string]Destination
	popularity   map[string]Popularity
	now          func() time.Time
}

// NewMemStore builds an empty in-memory store.
func NewMemStore() *MemStore {
	return &MemStore{
		routes:       make(map[string]conn.TransportQuote),
		fx:           make(map[string]conn.FxRate),
		destinations: make(map[string]Destination),
		popularity:   make(map[string]Popularity),
		now:          time.Now,
	}
}

func key(parts ...string) string {
	s := ""
	for i, p := range parts {
		if i > 0 {
			s += "|"
		}
		s += p
	}
	return s
}

// PutRoute stores a transport quote.
func (s *MemStore) PutRoute(_ context.Context, tenantID string, q conn.TransportQuote) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.routes[key(tenantID, q.From, q.To, string(q.Mode))] = q
	return nil
}

// GetRoute returns a fresh transport quote if present.
func (s *MemStore) GetRoute(_ context.Context, tenantID, from, to string, mode conn.TransportMode) (conn.TransportQuote, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	q, ok := s.routes[key(tenantID, from, to, string(mode))]
	if !ok || !q.Fresh(s.now()) {
		return conn.TransportQuote{}, false, nil
	}
	return q, true, nil
}

// PutFx stores an FX rate.
func (s *MemStore) PutFx(_ context.Context, tenantID string, r conn.FxRate) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fx[key(tenantID, r.Base, r.Quote)] = r
	return nil
}

// GetFx returns a fresh FX rate if present.
func (s *MemStore) GetFx(_ context.Context, tenantID, base, quote string) (conn.FxRate, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.fx[key(tenantID, base, quote)]
	if !ok || !r.Fresh(s.now()) {
		return conn.FxRate{}, false, nil
	}
	return r, true, nil
}

// UpsertDestination inserts or replaces a destination.
func (s *MemStore) UpsertDestination(_ context.Context, tenantID string, d Destination) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.destinations[key(tenantID, d.Country, d.City)] = d
	return nil
}

// ListDestinations returns all destinations for a tenant.
func (s *MemStore) ListDestinations(_ context.Context, tenantID string) ([]Destination, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []Destination
	for k, d := range s.destinations {
		if hasTenantPrefix(k, tenantID) {
			out = append(out, d)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Country != out[j].Country {
			return out[i].Country < out[j].Country
		}
		return out[i].City < out[j].City
	})
	return out, nil
}

// RecordVisit increments the aggregate hit count for a destination.
func (s *MemStore) RecordVisit(_ context.Context, tenantID, destination, kind string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := key(tenantID, destination, kind)
	p := s.popularity[k]
	p.Destination = destination
	p.Kind = kind
	p.Hits++
	p.UpdatedAt = s.now()
	s.popularity[k] = p
	return nil
}

// TopDestinations returns the most-visited destinations, highest first.
func (s *MemStore) TopDestinations(_ context.Context, tenantID string, limit int) ([]Popularity, error) {
	s.mu.RLock()
	var out []Popularity
	for k, p := range s.popularity {
		if hasTenantPrefix(k, tenantID) {
			out = append(out, p)
		}
	}
	s.mu.RUnlock()

	sort.Slice(out, func(i, j int) bool {
		if out[i].Hits != out[j].Hits {
			return out[i].Hits > out[j].Hits
		}
		return out[i].Destination < out[j].Destination // stable tie-break
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func hasTenantPrefix(k, tenantID string) bool {
	return len(k) > len(tenantID) && k[:len(tenantID)] == tenantID && k[len(tenantID)] == '|'
}

var _ Store = (*MemStore)(nil)
