package connectors

import (
	"math"
	"strings"
)

// LatLng is a geographic coordinate.
type LatLng struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// Geocoder resolves a city name/code to coordinates. A connector that needs
// coordinates (distance estimator, Overpass POI) depends on this interface, so
// the coordinate source can grow (static seed now, the knowledge base later)
// without touching the connector.
type Geocoder interface {
	Coords(city string) (LatLng, bool)
}

// StaticGeocoder is a case-insensitive in-memory Geocoder seeded from a map.
type StaticGeocoder map[string]LatLng

// Coords looks up a city (case-insensitive, trimmed).
func (g StaticGeocoder) Coords(city string) (LatLng, bool) {
	ll, ok := g[strings.ToLower(strings.TrimSpace(city))]
	return ll, ok
}

// NewStaticGeocoder builds a StaticGeocoder with lower-cased keys.
func NewStaticGeocoder(m map[string]LatLng) StaticGeocoder {
	g := make(StaticGeocoder, len(m))
	for k, v := range m {
		g[strings.ToLower(strings.TrimSpace(k))] = v
	}
	return g
}

// MergedGeocoder tries each geocoder in order and returns the first hit, so a
// per-request geocoder (e.g. coordinates the client supplied from autocomplete)
// can take precedence over a static seed.
type MergedGeocoder []Geocoder

// Coords returns the first geocoder's hit.
func (m MergedGeocoder) Coords(city string) (LatLng, bool) {
	for _, g := range m {
		if g == nil {
			continue
		}
		if ll, ok := g.Coords(city); ok {
			return ll, true
		}
	}
	return LatLng{}, false
}

// HaversineKm returns the great-circle distance between two points in km.
func HaversineKm(a, b LatLng) float64 {
	const r = 6371.0
	dLat := (b.Lat - a.Lat) * math.Pi / 180
	dLng := (b.Lng - a.Lng) * math.Pi / 180
	la1 := a.Lat * math.Pi / 180
	la2 := b.Lat * math.Pi / 180
	h := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(la1)*math.Cos(la2)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * r * math.Asin(math.Sqrt(h))
}
