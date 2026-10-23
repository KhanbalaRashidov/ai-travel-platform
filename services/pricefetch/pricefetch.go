// Package pricefetch turns several public web sources into a single, cheapest,
// provenance-tagged price for a transport leg, without any API token.
//
// Pipeline: fetch each source (server-side, realistic UA, graceful skip on
// anti-bot/non-2xx) -> ask the LLM (Groq, JSON mode) to EXTRACT prices from the
// fetched text -> GUARDRAIL: keep only amounts that appear verbatim in that text
// (the model may never invent a number) -> normalize currencies via FX and pick
// the cheapest -> cache the result (hot KV) so repeat queries skip the network.
//
// ToS: this package never ships scrapers of sources whose terms forbid it and
// never defeats anti-bot measures — a source that returns 403/429/challenge is
// skipped gracefully. Enabling a concrete source is a deliberate, ToS-vetted
// choice (see PriceSource.ToS). With no sources (the default) a caller falls
// back to the labeled estimate, so the platform runs token-less out of the box.
package pricefetch

import (
	"time"

	conn "github.com/KhanbalaRashidov/ai-travel-platform/services/connectors"
)

// LegQuery is everything a source needs to look up one leg.
type LegQuery struct {
	From       string
	To         string
	FromLat    float64
	FromLng    float64
	ToLat      float64
	ToLng      float64
	Mode       conn.TransportMode
	DepartDate string // YYYY-MM-DD, optional
	Adults     int
}

// Candidate is one source's price for a leg, kept with provenance. The detail
// fields (operator, times, baggage…) are optional: filled only when a source's
// page actually states them, never invented.
type Candidate struct {
	Source    string     `json:"source"`
	URL       string     `json:"url,omitempty"`
	Price     conn.Money `json:"price"`
	FetchedAt time.Time  `json:"fetched_at"`

	Operator string `json:"operator,omitempty"` // airline / bus / rail operator
	Depart   string `json:"depart,omitempty"`   // departure time, as printed
	Duration string `json:"duration,omitempty"` // total trip duration, as printed
	Stops    string `json:"stops,omitempty"`    // "direct" / "1 stop" …
	Baggage  string `json:"baggage,omitempty"`  // included/extra + size/weight
}

// Converter normalizes currencies (planner.StaticConverter satisfies it).
type Converter interface {
	Convert(amount float64, from, to string) (float64, error)
}
