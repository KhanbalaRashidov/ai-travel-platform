package ai

import "math"

// priceTolerance is the allowed absolute difference when matching a text price
// against a grounded value (covers rounding noise).
const priceTolerance = 0.01

// Violation is a price in the text that is not backed by any tool result.
type Violation struct {
	Value float64 `json:"value"`
}

// Validate is the price-hallucination guardrail. It returns every currency
// amount in text that is NOT present in the grounded set (the values the tools
// actually returned). An empty result means the text is fully grounded.
//
// Pure and deterministic — the single most important safety check in the AI
// path, so it is unit-tested directly.
func Validate(text string, grounded []float64) []Violation {
	var violations []Violation
	for _, v := range ExtractMoneyValues(text) {
		if !containsApprox(grounded, v) {
			violations = append(violations, Violation{Value: v})
		}
	}
	return violations
}

func containsApprox(set []float64, v float64) bool {
	for _, g := range set {
		if math.Abs(g-v) <= priceTolerance {
			return true
		}
	}
	return false
}
