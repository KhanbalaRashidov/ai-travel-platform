// Package connectors fetches external travel data (flights, ground transport,
// hotels, POIs, FX) from open/free sources with a scrape/estimate fallback,
// normalizing everything with provenance (source + fetched_at). It never
// depends on a single gated API: each capability is a fallback chain of
// sources, and a last-resort labeled estimate keeps the system responsive.
//
// Split-ready: this package can later become its own service (E2).
package connectors
