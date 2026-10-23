// Package connectors will fetch external travel data (flights, ground transport,
// hotels, POIs, FX) from open/free sources with a scrape fallback, normalizing
// everything with source + fetched_at. Not dependent on gated APIs.
//
// Split-ready: implemented in E2.
package connectors

// Source is the common interface every data connector will implement (E2).
type Source interface {
	// Name returns the connector's identifier.
	Name() string
}
