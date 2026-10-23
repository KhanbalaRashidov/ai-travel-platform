package planner

// ObjectiveComparison contrasts the cost-optimal route with the
// comfort-optimal one (shortest total travel time). Comfort.Cost is the MONEY
// cost of the comfort order evaluated on the cost matrix, so the UI can show
// how much money the cheapest order saves versus the most convenient one.
type ObjectiveComparison struct {
	CostOptimal Route   `json:"cost_optimal"`
	Comfort     Route   `json:"comfort"`
	MoneySaved  float64 `json:"money_saved"`
}

// CompareObjectives sequences the cities two ways — cheapest (costM) and most
// comfortable (durationM) — and reports the money saved by choosing cost over
// comfort. Reuses the same fixed-endpoint TSP for both objectives.
func CompareObjectives(costM, durationM Matrix, start, end int) (ObjectiveComparison, error) {
	costBest, _, err := Optimize(costM, start, end)
	if err != nil {
		return ObjectiveComparison{}, err
	}
	durBest, _, err := Optimize(durationM, start, end)
	if err != nil {
		return ObjectiveComparison{}, err
	}

	comfortMoney := round2(pathCost(durBest.Order, costM, durBest.Variant == RoundTrip))
	comfort := Route{Variant: durBest.Variant, Order: durBest.Order, Cost: comfortMoney}

	return ObjectiveComparison{
		CostOptimal: costBest,
		Comfort:     comfort,
		MoneySaved:  round2(comfortMoney - costBest.Cost),
	}, nil
}
