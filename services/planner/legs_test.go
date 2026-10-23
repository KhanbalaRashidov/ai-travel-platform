package planner

import "testing"

func TestCompareLeg_CheapestAndFastest(t *testing.T) {
	opts := []LegOption{
		{Mode: ModeFlight, Price: Money{Amount: 120, Currency: "EUR"}, DurationMinutes: 90},
		{Mode: ModeTrain, Price: Money{Amount: 40, Currency: "EUR"}, DurationMinutes: 600},
		{Mode: ModeBus, Price: Money{Amount: 25, Currency: "EUR"}, DurationMinutes: 720},
	}
	cmp, err := CompareLeg("BAK", "TBS", opts)
	if err != nil {
		t.Fatal(err)
	}
	if cmp.Cheapest.Mode != ModeBus || cmp.Cheapest.Price.Amount != 25 {
		t.Fatalf("cheapest wrong: %+v", cmp.Cheapest)
	}
	if cmp.Fastest.Mode != ModeFlight || cmp.Fastest.DurationMinutes != 90 {
		t.Fatalf("fastest wrong: %+v", cmp.Fastest)
	}
	// options sorted cheapest-first
	if cmp.Options[0].Mode != ModeBus || cmp.Options[2].Mode != ModeFlight {
		t.Fatalf("not sorted by price: %+v", cmp.Options)
	}
}

func TestCompareLeg_Empty(t *testing.T) {
	if _, err := CompareLeg("A", "B", nil); err != ErrNoOptions {
		t.Fatalf("expected ErrNoOptions, got %v", err)
	}
}

func TestCompareLegs_PropagatesError(t *testing.T) {
	legs := []LegInput{
		{From: "A", To: "B", Options: []LegOption{{Mode: ModeBus, Price: Money{Amount: 10, Currency: "EUR"}, DurationMinutes: 100}}},
		{From: "B", To: "C", Options: nil},
	}
	if _, err := CompareLegs(legs); err != ErrNoOptions {
		t.Fatalf("expected ErrNoOptions, got %v", err)
	}
}
