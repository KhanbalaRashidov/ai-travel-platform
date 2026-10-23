// Package planner will build the cost-optimal multi-country roadmap:
// TSP-style city sequencing + per-leg flight/train/bus comparison +
// day allocation + hotel tier selection + a deterministic live total.
//
// Split-ready: this package can later become its own service. Implemented in E3.
package planner

// Service is the planning entry point. Concrete implementation lands in E3.
type Service interface {
	// Plan is a placeholder for the future planning API.
}
