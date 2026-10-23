package planner

import (
	"sort"
	"strings"

	conn "github.com/KhanbalaRashidov/ai-travel-platform/services/connectors"
)

// Reuse the normalized connector types so the engine and the data layer speak
// the same language.
type (
	Money         = conn.Money
	TransportMode = conn.TransportMode
)

// Re-exported transport modes for convenience.
const (
	ModeFlight = conn.ModeFlight
	ModeTrain  = conn.ModeTrain
	ModeBus    = conn.ModeBus
)

// ComfortTier is the user's comfort level; it maps to a star band.
type ComfortTier string

const (
	TierBudget   ComfortTier = "budget"
	TierStandard ComfortTier = "standard"
	TierVIP      ComfortTier = "vip"
)

// TierStars returns the [min,max] hotel star band for a comfort tier.
func TierStars(t ComfortTier) (min, max int) {
	switch t {
	case TierBudget:
		return 2, 3
	case TierStandard:
		return 3, 4
	case TierVIP:
		return 4, 5
	default:
		return 2, 5
	}
}

// PricedLeg is a chosen transport option for one city->city hop.
type PricedLeg struct {
	From  int           `json:"from"`
	To    int           `json:"to"`
	Mode  TransportMode `json:"mode"`
	Price Money         `json:"price"`
}

// PricedHotel is a chosen hotel for one city (tier-driven).
type PricedHotel struct {
	City    int   `json:"city"`
	Stars   int   `json:"stars"`
	Nights  int   `json:"nights"`
	Nightly Money `json:"nightly"`
}

// PlanInput is everything the deterministic pricer needs. The Order is fixed by
// the sequencer; changing tiers/modes only changes Legs/Hotels, never Order.
type PlanInput struct {
	Order           []int         `json:"order"`
	Legs            []PricedLeg   `json:"legs"`
	Hotels          []PricedHotel `json:"hotels"`
	ActivitiesTotal Money         `json:"activities_total"`
	DisplayCurrency string        `json:"display_currency"`
}

// Breakdown is the computed cost, all converted to DisplayCurrency.
type Breakdown struct {
	Transport  float64 `json:"transport"`
	Hotels     float64 `json:"hotels"`
	Activities float64 `json:"activities"`
	Total      float64 `json:"total"`
	Currency   string  `json:"currency"`
}

// CitySetKey builds an order-independent key for a set of city identifiers.
// Sequencing is cached on this key, so tier/mode toggles (same set) never
// trigger a re-sequence.
func CitySetKey(cities []string) string {
	cp := append([]string(nil), cities...)
	sort.Strings(cp)
	return strings.Join(cp, "|")
}
