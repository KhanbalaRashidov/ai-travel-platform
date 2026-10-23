package knowledge

import (
	"context"
	"time"

	conn "github.com/KhanbalaRashidov/ai-travel-platform/services/connectors"
)

// Destination is a place the system has learned about.
type Destination struct {
	Country         string  `json:"country"`
	City            string  `json:"city"`
	Lat             float64 `json:"lat"`
	Lng             float64 `json:"lng"`
	Summary         string  `json:"summary"`
	conn.Provenance `json:"provenance"`
}

// Popularity is an aggregate visit count for a destination (privacy-safe).
type Popularity struct {
	Destination string    `json:"destination"`
	Kind        string    `json:"kind"` // e.g. "city", "country"
	Hits        int64     `json:"hits"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Recommendation is a dynamic suggestion derived from the knowledge base.
type Recommendation struct {
	Destination string `json:"destination"`
	Hits        int64  `json:"hits"`
	Reason      string `json:"reason"`
}

// Store is the knowledge-base persistence port. Lookups return ok=false for
// missing OR stale cache entries so callers transparently re-fetch.
type Store interface {
	PutRoute(ctx context.Context, tenantID string, q conn.TransportQuote) error
	GetRoute(ctx context.Context, tenantID, from, to string, mode conn.TransportMode) (conn.TransportQuote, bool, error)

	PutFx(ctx context.Context, tenantID string, r conn.FxRate) error
	GetFx(ctx context.Context, tenantID, base, quote string) (conn.FxRate, bool, error)

	UpsertDestination(ctx context.Context, tenantID string, d Destination) error
	ListDestinations(ctx context.Context, tenantID string) ([]Destination, error)

	RecordVisit(ctx context.Context, tenantID, destination, kind string) error
	TopDestinations(ctx context.Context, tenantID string, limit int) ([]Popularity, error)
}
