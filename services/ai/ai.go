// Package ai is the LLM orchestrator. It extracts trip constraints from user
// input, calls tools (connectors + planner + knowledge) for REAL data, and
// writes the itinerary narrative — but it never invents prices. Every number in
// the final text is validated against the values the tools actually returned
// (grounding); ungrounded prices are rejected and regenerated.
//
// The LLM sits behind an interface so it can be swapped (Sonnet/Haiku/Gemini)
// and mocked in tests, so this whole package builds and tests without a network
// or Docker. Split-ready (E4).
package ai
