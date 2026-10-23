// Package planner is the cost-optimization engine. It separates two concerns
// so the UI can re-price instantly:
//
//   - Sequencing (expensive): order cities to minimize transport cost as a
//     fixed-start, optional-fixed-end TSP, evaluating both a round-trip
//     (start==end) and an open-jaw (start!=end) variant and keeping the cheaper
//     as best + the other as an alternative. Recomputed ONLY when the city set
//     changes (see Sequencer's cache).
//   - Pricing (cheap, deterministic): given a fixed order, the chosen per-leg
//     modes, hotel tier/stars, nights and FX, compute the total breakdown.
//     Re-run on every tier/star/mode toggle WITHOUT re-sequencing.
//
// All math is deterministic Go (never an LLM), so the live total is exact and
// fast. Split-ready (E3).
package planner
