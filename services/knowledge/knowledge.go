// Package knowledge is the self-growing knowledge base. It is a write-through
// cache of provenance-tagged facts (routes, FX, destinations) that the system
// learns as users plan trips: data found once is stored and reused, missing
// data is fetched (connectors/AI) then saved, and aggregate popularity powers
// dynamic recommendations ("travelers like you chose ...").
//
// A Store has an in-memory implementation (tests / Docker-free dev) and a
// Postgres implementation (production, backed by the 0002_knowledge tables).
// Split-ready (E6).
package knowledge
