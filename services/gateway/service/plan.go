package service

import "github.com/KhanbalaRashidov/ai-travel-platform/services/planner"

// PlanService exposes the cost-optimization engine. It is pure/stateless (no
// DB), so the re-pricing path can be called on every UI toggle.
type PlanService struct{}

// NewPlanService builds the plan service.
func NewPlanService() *PlanService { return &PlanService{} }

// Sequence orders the cities (round-trip vs open-jaw) for a cost matrix.
func (s *PlanService) Sequence(m planner.Matrix, start, end int) (best, alt planner.Route, err error) {
	return planner.Optimize(m, start, end)
}

// Price re-computes the deterministic cost breakdown for a fixed order.
func (s *PlanService) Price(in planner.PlanInput, rates map[string]float64) (planner.Breakdown, error) {
	return planner.Price(in, planner.StaticConverter{Rates: rates})
}

// CompareLegs returns the cheapest/fastest option + sorted alternatives per leg.
func (s *PlanService) CompareLegs(legs []planner.LegInput) ([]planner.LegComparison, error) {
	return planner.CompareLegs(legs)
}
