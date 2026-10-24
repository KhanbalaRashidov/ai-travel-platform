package ai

import (
	"context"
	"testing"
)

func TestSuggestCountryDestinations_ParsesAndStripsRatings(t *testing.T) {
	// The model returns a rating; the guardrail must strip it (no real source).
	llm := &MockLLM{Script: []Completion{{Content: "```json\n" + `{
		"countries":[
			{"country":"Georgia","cities":[
				{"city":"Tbilisi","reason":"Old town + sulphur baths","best_season":"spring","rating":4.8,"review_count":1200},
				{"city":"Batumi","reason":"Black Sea resort","best_season":"summer"}
			]}
		],
		"budget_advice":"3-4 days, bus between cities, 3-star hotels."
	}` + "\n```"}}}

	got, err := SuggestCountryDestinations(context.Background(), llm, []string{"Georgia"}, 800, "EUR", "English")
	if err != nil {
		t.Fatalf("suggest: %v", err)
	}
	if len(got.Countries) != 1 || got.Countries[0].Country != "Georgia" {
		t.Fatalf("country not parsed: %+v", got.Countries)
	}
	cities := got.Countries[0].Cities
	if len(cities) != 2 || cities[0].City != "Tbilisi" {
		t.Fatalf("cities not parsed: %+v", cities)
	}
	// Guardrail: rating/review must be stripped even though the model sent them.
	if cities[0].Rating != 0 || cities[0].ReviewCount != 0 {
		t.Fatalf("ratings must be stripped (no real source): %+v", cities[0])
	}
	if cities[0].Reason == "" || cities[0].BestSeason == "" {
		t.Fatalf("reason/season should be kept: %+v", cities[0])
	}
	if cities[0].Country != "Georgia" {
		t.Fatalf("city country should default to its group: %+v", cities[0])
	}
	if got.BudgetAdvice == "" {
		t.Fatal("budget advice should be present when a budget was given")
	}
}

func TestSuggestCountryDestinations_NoLLMOrCountries(t *testing.T) {
	if _, err := SuggestCountryDestinations(context.Background(), nil, []string{"Italy"}, 0, "EUR", "English"); err == nil {
		t.Fatal("expected an error without an LLM")
	}
	llm := &MockLLM{Script: []Completion{{Content: "{}"}}}
	if _, err := SuggestCountryDestinations(context.Background(), llm, nil, 0, "EUR", "English"); err == nil {
		t.Fatal("expected an error with no countries")
	}
}
