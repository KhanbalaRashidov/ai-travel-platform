package planner

import "testing"

func TestCompareObjectives(t *testing.T) {
	// 3 cities. Cheapest order differs from fastest order.
	// cost: 0->2 cheap via 1; direct 0->2 pricey.
	costM := Matrix{
		{0, 1, 9},
		{1, 0, 1},
		{9, 1, 0},
	}
	// duration: direct 0->2 is quick; going via 1 is slow.
	durM := Matrix{
		{0, 50, 10},
		{50, 0, 50},
		{10, 50, 0},
	}
	cmp, err := CompareObjectives(costM, durM, 0, -1)
	if err != nil {
		t.Fatal(err)
	}
	// cost-optimal should route through node 1 (cheap edges)
	if cmp.CostOptimal.Cost != 2 {
		t.Fatalf("expected cost-optimal cost 2, got %v (order %v)", cmp.CostOptimal.Cost, cmp.CostOptimal.Order)
	}
	// comfort order's money cost must be >= cost-optimal, so savings >= 0
	if cmp.MoneySaved < 0 {
		t.Fatalf("money saved should be >= 0, got %v", cmp.MoneySaved)
	}
	if len(cmp.Comfort.Order) != 3 {
		t.Fatalf("comfort order should visit all 3, got %v", cmp.Comfort.Order)
	}
}
