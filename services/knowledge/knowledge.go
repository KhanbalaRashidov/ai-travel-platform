// Package knowledge is the self-growing knowledge base: it persists everything
// found (destinations, costs, routes, POIs, FX) with source/fetched_at/ttl,
// reuses it for future users, powers dynamic recommendations, and maintains a
// popular-destinations ranking.
//
// Split-ready: implemented in E6.
package knowledge

// Store is the knowledge-base entry point. Concrete implementation lands in E6.
type Store interface {
	// Store is a placeholder for the future knowledge-base API.
}
