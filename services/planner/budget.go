package planner

// RelaxKind is a type of budget relaxation the solver can suggest.
type RelaxKind string

const (
	RelaxLowerTier        RelaxKind = "lower_tier"
	RelaxCheaperTransport RelaxKind = "cheaper_transport"
	RelaxFewerNights      RelaxKind = "fewer_nights"
	RelaxFewerCities      RelaxKind = "fewer_cities"
)

// Suggestion is one actionable relaxation.
type Suggestion struct {
	Kind   RelaxKind `json:"kind"`
	Detail string    `json:"detail"`
}

// BudgetResult is the outcome of fitting a breakdown to a budget.
type BudgetResult struct {
	Total       float64      `json:"total"`
	Budget      float64      `json:"budget"`
	Fits        bool         `json:"fits"`
	Over        float64      `json:"over"`
	Suggestions []Suggestion `json:"suggestions,omitempty"`
}

// FitBudget checks a breakdown against a budget. When over, it proposes
// relaxations ordered by the category contributing most to the cost, so the
// UI can nudge the user toward a feasible plan. Objective: minimize total
// subject to budget; this reports feasibility + the cheapest levers to pull.
func FitBudget(b Breakdown, budget float64) BudgetResult {
	res := BudgetResult{Total: b.Total, Budget: budget}
	if budget <= 0 || b.Total <= budget {
		res.Fits = true
		return res
	}
	res.Over = round2(b.Total - budget)

	if b.Hotels >= b.Transport {
		res.Suggestions = []Suggestion{
			{RelaxLowerTier, "lower hotel tier / stars"},
			{RelaxFewerNights, "reduce nights per city"},
			{RelaxCheaperTransport, "use cheaper transport modes"},
		}
	} else {
		res.Suggestions = []Suggestion{
			{RelaxCheaperTransport, "use cheaper transport modes"},
			{RelaxFewerCities, "drop the most expensive-to-reach city"},
			{RelaxLowerTier, "lower hotel tier / stars"},
		}
	}
	return res
}
