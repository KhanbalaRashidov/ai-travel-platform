package knowledge

import (
	"context"
	"time"

	conn "github.com/KhanbalaRashidov/ai-travel-platform/services/connectors"
)

// StarterDestinations is the small curated set the platform ships with, so the
// first user sees real cities before the knowledge base has grown from traffic.
// It is also the coordinate seed for the distance estimator and POI connector.
func StarterDestinations() []Destination {
	now := time.Now().UTC()
	prov := conn.Provenance{Source: "seed", FetchedAt: now} // curated, no expiry
	mk := func(country, city string, lat, lng float64, summary string) Destination {
		return Destination{Country: country, City: city, Lat: lat, Lng: lng, Summary: summary, Provenance: prov}
	}
	return []Destination{
		mk("Azerbaijan", "Baku", 40.4093, 49.8671, "Caspian capital: Old City, Flame Towers, seaside boulevard."),
		mk("Georgia", "Tbilisi", 41.7151, 44.8271, "Old town, sulfur baths, wine and mountain day-trips."),
		mk("Georgia", "Batumi", 41.6168, 41.6367, "Black Sea resort city with a lively promenade."),
		mk("Armenia", "Yerevan", 40.1792, 44.4991, "Cascade, Republic Square, ancient monasteries nearby."),
		mk("Turkey", "Istanbul", 41.0082, 28.9784, "Two continents: Hagia Sophia, bazaars, Bosphorus."),
	}
}

// SeedGeocoder builds a Geocoder from the starter set for connectors that need
// coordinates (distance estimator, Overpass POI).
func SeedGeocoder() conn.Geocoder {
	m := make(map[string]conn.LatLng)
	for _, d := range StarterDestinations() {
		m[d.City] = conn.LatLng{Lat: d.Lat, Lng: d.Lng}
	}
	return conn.NewStaticGeocoder(m)
}

// Seed upserts the starter destinations for a tenant the first time (no-op if
// the tenant already has destinations). Safe to call on every boot.
func Seed(ctx context.Context, store Store, tenantID string) error {
	existing, err := store.ListDestinations(ctx, tenantID)
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		return nil
	}
	for _, d := range StarterDestinations() {
		if err := store.UpsertDestination(ctx, tenantID, d); err != nil {
			return err
		}
	}
	return nil
}
