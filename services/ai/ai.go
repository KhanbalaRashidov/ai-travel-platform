// Package ai is the LLM orchestrator: it extracts constraints from user input,
// calls tools (connectors + planner + knowledge) for real data, writes the
// day-by-day narrative, and enforces guardrails so prices are never invented.
//
// Split-ready: implemented in E4.
package ai

// Orchestrator is the AI entry point. Concrete implementation lands in E4.
type Orchestrator interface {
	// Orchestrator is a placeholder for the future AI API.
}
