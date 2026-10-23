package connectors

import (
	"context"
	"errors"
)

// ErrUnavailable signals a source could not provide data; it triggers the next
// source in a fallback chain.
var ErrUnavailable = errors.New("source unavailable")

// Source is the base every connector implements.
type Source interface {
	Name() string
}

// TransportSource provides per-leg transport quotes.
type TransportSource interface {
	Source
	Transport(ctx context.Context, from, to string, mode TransportMode) (TransportQuote, error)
}

// HotelSource provides nightly hotel quotes by star tier.
type HotelSource interface {
	Source
	Hotel(ctx context.Context, city string, stars int) (HotelQuote, error)
}

// POISource provides points of interest for a city.
type POISource interface {
	Source
	POIs(ctx context.Context, city string, limit int) ([]POI, error)
}

// FxSource provides currency conversion rates.
type FxSource interface {
	Source
	Rate(ctx context.Context, base, quote string) (FxRate, error)
}
