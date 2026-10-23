package planner

import "testing"

// square: unit-edge 4-cycle (0-1-2-3-0) with cost-2 diagonals.
var square = Matrix{
	{0, 1, 2, 1},
	{1, 0, 1, 2},
	{2, 1, 0, 1},
	{1, 2, 1, 0},
}

// asym: makes the return home expensive, so open-jaw beats round-trip.
var asym = Matrix{
	{0, 1, 5},
	{1, 0, 1},
	{100, 100, 0},
}

func assertPerm(t *testing.T, order []int, n, start int) {
	t.Helper()
	if len(order) != n {
		t.Fatalf("order length = %d, want %d", len(order), n)
	}
	if order[0] != start {
		t.Fatalf("order must start at %d, got %d", start, order[0])
	}
	seen := make(map[int]bool)
	for _, v := range order {
		if v < 0 || v >= n || seen[v] {
			t.Fatalf("invalid/duplicate node %d in %v", v, order)
		}
		seen[v] = true
	}
}

func TestExactRoundTrip_Square(t *testing.T) {
	r, err := Solve(square, 0, -1, RoundTrip)
	if err != nil {
		t.Fatal(err)
	}
	if r.Cost != 4 {
		t.Fatalf("round-trip cost = %v, want 4", r.Cost)
	}
	assertPerm(t, r.Order, 4, 0)
}

func TestOptimize_OpenJawBeatsRoundTrip(t *testing.T) {
	best, alt, err := Optimize(asym, 0, -1)
	if err != nil {
		t.Fatal(err)
	}
	if best.Variant != OpenJaw || best.Cost != 2 {
		t.Fatalf("best = %+v, want open_jaw cost 2", best)
	}
	if alt.Variant != RoundTrip || alt.Cost != 102 {
		t.Fatalf("alt = %+v, want round_trip cost 102", alt)
	}
}

func TestSolve_FixedEnd(t *testing.T) {
	r, err := Solve(asym, 0, 1, OpenJaw) // must end at node 1
	if err != nil {
		t.Fatal(err)
	}
	if r.Order[len(r.Order)-1] != 1 {
		t.Fatalf("path must end at 1, got %v", r.Order)
	}
	if r.Cost != 105 { // 0->2 (5) + 2->1 (100)
		t.Fatalf("fixed-end cost = %v, want 105", r.Cost)
	}
	assertPerm(t, r.Order, 3, 0)
}

func TestHeuristic_MatchesExactOnSquare(t *testing.T) {
	order, cost, err := heuristic(square, 0, -1, true)
	if err != nil {
		t.Fatal(err)
	}
	if cost != 4 {
		t.Fatalf("heuristic round-trip cost = %v, want 4", cost)
	}
	assertPerm(t, order, 4, 0)
}

func TestHeuristic_RespectsFixedEnd(t *testing.T) {
	order, _, err := heuristic(asym, 0, 1, false)
	if err != nil {
		t.Fatal(err)
	}
	if order[0] != 0 || order[len(order)-1] != 1 {
		t.Fatalf("expected start 0 end 1, got %v", order)
	}
}
