package planner

import "testing"

func TestFitBudget_Within(t *testing.T) {
	r := FitBudget(Breakdown{Transport: 50, Hotels: 40, Total: 100}, 150)
	if !r.Fits || r.Over != 0 || len(r.Suggestions) != 0 {
		t.Fatalf("expected fit with no suggestions, got %+v", r)
	}
}

func TestFitBudget_OverHotelsDominant(t *testing.T) {
	r := FitBudget(Breakdown{Transport: 50, Hotels: 120, Total: 170}, 100)
	if r.Fits || r.Over != 70 {
		t.Fatalf("expected over by 70, got %+v", r)
	}
	if len(r.Suggestions) == 0 || r.Suggestions[0].Kind != RelaxLowerTier {
		t.Fatalf("expected lower_tier first, got %+v", r.Suggestions)
	}
}

func TestFitBudget_OverTransportDominant(t *testing.T) {
	r := FitBudget(Breakdown{Transport: 200, Hotels: 50, Total: 250}, 100)
	if r.Fits || r.Suggestions[0].Kind != RelaxCheaperTransport {
		t.Fatalf("expected cheaper_transport first, got %+v", r)
	}
}
