// Package estimate is a key-less, offline "cold source" of last resort: it
// derives indicative transport and hotel prices from distance and a star table.
// Every datum it returns is provenance-tagged Estimated=true, so the UI can mark
// it clearly and a real source always wins in a fallback chain. This keeps the
// product functional without any gated/paid API while never passing an estimate
// off as a quoted price.
package estimate

import (
	"context"
	"math"
	"time"

	conn "github.com/KhanbalaRashidov/ai-travel-platform/services/connectors"
)

const sourceName = "distance-estimate"

// Estimator implements TransportSource and HotelSource from coordinates.
type Estimator struct {
	geo conn.Geocoder
	ttl time.Duration
	now func() time.Time
}

// New builds an Estimator. ttl sets how long an estimate stays fresh in caches
// (<=0 means a 7-day default). geo resolves city coordinates.
func New(geo conn.Geocoder, ttl time.Duration) *Estimator {
	if ttl <= 0 {
		ttl = 7 * 24 * time.Hour
	}
	return &Estimator{geo: geo, ttl: ttl, now: time.Now}
}

// Name identifies the source for provenance.
func (e *Estimator) Name() string { return sourceName }

func (e *Estimator) provenance() conn.Provenance {
	now := e.now().UTC()
	return conn.Provenance{
		Source:    sourceName,
		FetchedAt: now,
		ExpiresAt: now.Add(e.ttl),
		Estimated: true,
	}
}

// Transport returns an estimated quote for a leg, or ErrUnavailable when either
// endpoint is unknown or the mode is not viable for the distance.
func (e *Estimator) Transport(_ context.Context, from, to string, mode conn.TransportMode) (conn.TransportQuote, error) {
	a, okA := e.geo.Coords(from)
	b, okB := e.geo.Coords(to)
	if !okA || !okB {
		return conn.TransportQuote{}, conn.ErrUnavailable
	}
	km := conn.HaversineKm(a, b)
	price, dur, ok := estimateLeg(km, mode)
	if !ok {
		return conn.TransportQuote{}, conn.ErrUnavailable
	}
	return conn.TransportQuote{
		From:       from,
		To:         to,
		Mode:       mode,
		Price:      conn.Money{Amount: price, Currency: "EUR"},
		Duration:   dur,
		Provenance: e.provenance(),
	}, nil
}

// estimateLeg maps distance+mode to an indicative price (EUR) and duration.
// Returns ok=false when the mode is not viable for the distance. Pure.
func estimateLeg(km float64, mode conn.TransportMode) (float64, time.Duration, bool) {
	switch mode {
	case conn.ModeFlight:
		price := math.Round(30 + 0.08*km)
		// ~1h overhead + 800 km/h cruise.
		dur := time.Duration((1 + km/800) * float64(time.Hour))
		return price, dur, true
	case conn.ModeTrain:
		if km > 1500 {
			return 0, 0, false
		}
		price := math.Round(0.05 * km)
		dur := time.Duration((km / 90) * float64(time.Hour)) // ~90 km/h avg
		return price, dur, true
	case conn.ModeBus:
		if km > 1200 {
			return 0, 0, false
		}
		price := math.Round(0.03 * km)
		dur := time.Duration((km / 65) * float64(time.Hour)) // ~65 km/h avg
		return price, dur, true
	}
	return 0, 0, false
}

// nightlyByStar is an indicative EUR nightly rate per star tier.
var nightlyByStar = map[int]float64{2: 35, 3: 55, 4: 90, 5: 150}

// Hotel returns an estimated nightly quote for a star tier in a city.
func (e *Estimator) Hotel(_ context.Context, city string, stars int) (conn.HotelQuote, error) {
	if stars < 2 {
		stars = 2
	}
	if stars > 5 {
		stars = 5
	}
	nightly, ok := nightlyByStar[stars]
	if !ok {
		return conn.HotelQuote{}, conn.ErrUnavailable
	}
	return conn.HotelQuote{
		City:       city,
		Stars:      stars,
		Nightly:    conn.Money{Amount: nightly, Currency: "EUR"},
		Provenance: e.provenance(),
	}, nil
}

var (
	_ conn.TransportSource = (*Estimator)(nil)
	_ conn.HotelSource     = (*Estimator)(nil)
)
