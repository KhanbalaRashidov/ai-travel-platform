package ai

import "math"

// TaskKind lets the orchestrator route work to the right model: expensive
// reasoning vs cheap bulk work (translation, formatting).
type TaskKind string

const (
	TaskReasoning TaskKind = "reasoning"
	TaskBulk      TaskKind = "bulk"
)

// ModelRoute is a model choice with its USD price per 1M tokens.
type ModelRoute struct {
	Name        string  `json:"name"`
	InputPer1M  float64 `json:"input_per_1m"`
	OutputPer1M float64 `json:"output_per_1m"`
}

// Default routes (indicative pricing; keep in sync with the provider).
var (
	ModelPrimary = ModelRoute{Name: "claude-sonnet-5-5", InputPer1M: 2, OutputPer1M: 10}
	ModelBulk    = ModelRoute{Name: "claude-haiku-5-5", InputPer1M: 0.10, OutputPer1M: 0.50}
)

// RouteModel sends bulk tasks to the cheap model and everything else to the
// primary reasoning model.
func RouteModel(kind TaskKind) ModelRoute {
	if kind == TaskBulk {
		return ModelBulk
	}
	return ModelPrimary
}

// EstimateCostUSD returns the token cost for a route (rounded to 6 decimals).
func EstimateCostUSD(r ModelRoute, inTokens, outTokens int) float64 {
	c := float64(inTokens)/1e6*r.InputPer1M + float64(outTokens)/1e6*r.OutputPer1M
	return math.Round(c*1e6) / 1e6
}

// Budget tracks spend against an optional cap (limit <= 0 means unlimited).
type Budget struct {
	LimitUSD float64 `json:"limit_usd"`
	SpentUSD float64 `json:"spent_usd"`
}

// NewBudget creates a budget with the given USD cap.
func NewBudget(limitUSD float64) *Budget { return &Budget{LimitUSD: limitUSD} }

// CanAfford reports whether cost fits under the remaining budget.
func (b *Budget) CanAfford(cost float64) bool {
	return b.LimitUSD <= 0 || b.SpentUSD+cost <= b.LimitUSD+1e-9
}

// Add charges cost if affordable and reports success. An unaffordable charge
// is rejected (fail closed) so a runaway loop cannot overspend.
func (b *Budget) Add(cost float64) bool {
	if !b.CanAfford(cost) {
		return false
	}
	b.SpentUSD = math.Round((b.SpentUSD+cost)*1e6) / 1e6
	return true
}

// Remaining returns the USD left (Inf for an unlimited budget).
func (b *Budget) Remaining() float64 {
	if b.LimitUSD <= 0 {
		return math.Inf(1)
	}
	if r := b.LimitUSD - b.SpentUSD; r > 0 {
		return r
	}
	return 0
}
