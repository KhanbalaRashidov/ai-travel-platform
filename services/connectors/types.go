package connectors

import "time"

// Money is an amount in a currency.
type Money struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

// Provenance is attached to every datum: where it came from, when, and whether
// it is a real source value or a labeled estimate. This is what makes the
// knowledge base trustworthy and lets the UI show "as of".
type Provenance struct {
	Source    string    `json:"source"`
	FetchedAt time.Time `json:"fetched_at"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
	Estimated bool      `json:"estimated"`
}

// Fresh reports whether the datum is still valid at the given time.
// A zero ExpiresAt means "no expiry".
func (p Provenance) Fresh(now time.Time) bool {
	return p.ExpiresAt.IsZero() || now.Before(p.ExpiresAt)
}

// TransportMode enumerates per-leg transport modes.
type TransportMode string

const (
	ModeFlight TransportMode = "flight"
	ModeTrain  TransportMode = "train"
	ModeBus    TransportMode = "bus"
	ModeFerry  TransportMode = "ferry"
)

// TransportQuote is an indicative price + duration for a city->city leg.
type TransportQuote struct {
	From       string        `json:"from"`
	To         string        `json:"to"`
	Mode       TransportMode `json:"mode"`
	Price      Money         `json:"price"`
	Duration   time.Duration `json:"duration"`
	DeepLink   string        `json:"deep_link,omitempty"`
	Provenance `json:"provenance"`
}

// HotelQuote is an indicative nightly price for a star tier in a city.
type HotelQuote struct {
	City       string `json:"city"`
	Stars      int    `json:"stars"`
	Nightly    Money  `json:"nightly"`
	DeepLink   string `json:"deep_link,omitempty"`
	Provenance `json:"provenance"`
}

// POI is a point of interest.
type POI struct {
	City     string  `json:"city"`
	Name     string  `json:"name"`
	Category string  `json:"category"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
	Image    string  `json:"image,omitempty"`   // free image URL (OSM image / Wikimedia Commons)
	Website  string  `json:"website,omitempty"` // official site, when tagged
	Wiki     string  `json:"wiki,omitempty"`    // Wikipedia article URL, when tagged
	Desc     string  `json:"desc,omitempty"`    // short description (AI/knowledge POIs)
	Free     bool    `json:"free,omitempty"`    // true = no entry fee
	Price    string  `json:"price,omitempty"`   // approximate "~"-marked entry price
	// Rating fields are populated ONLY from a real source (never invented); when
	// no source provides them they stay zero/empty (no fake stars).
	Rating        float64 `json:"rating,omitempty"`
	ReviewCount   int     `json:"review_count,omitempty"`
	ReviewSummary string  `json:"review_summary,omitempty"`
	Provenance    `json:"provenance"`
}

// FxRate is a conversion rate base -> quote (1 base = Rate quote).
type FxRate struct {
	Base       string  `json:"base"`
	Quote      string  `json:"quote"`
	Rate       float64 `json:"rate"`
	Provenance `json:"provenance"`
}
