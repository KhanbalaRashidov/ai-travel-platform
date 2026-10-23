package ai

import (
	"context"
	"testing"
)

func brief() PlanBrief {
	return PlanBrief{
		Cities:          []string{"Baku", "Tbilisi"},
		Legs:            []LegFact{{From: "Baku", To: "Tbilisi", Mode: "flight", Price: 120}},
		Nights:          2,
		Currency:        "EUR",
		TransportTotal:  120,
		HotelsTotal:     360,
		ActivitiesTotal: 80,
		Total:           560,
	}
}

func TestWriteItinerary_GroundedPasses(t *testing.T) {
	llm := &MockLLM{Script: []Completion{
		{Content: "Day 1: Baku. Fly to Tbilisi for €120. Hotels €360, activities €80. Total €560."},
	}}
	res, err := WriteItinerary(context.Background(), llm, brief(), 1)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if len(res.Violations) != 0 {
		t.Fatalf("grounded text should have no violations, got %+v", res.Violations)
	}
	if res.Steps != 1 || llm.Calls != 1 {
		t.Fatalf("should pass on first try, steps=%d calls=%d", res.Steps, llm.Calls)
	}
}

func TestWriteItinerary_HallucinationTriggersRegen(t *testing.T) {
	llm := &MockLLM{Script: []Completion{
		{Content: "Fly to Tbilisi for €999."},                                          // ungrounded price
		{Content: "Fly to Tbilisi for €120. Hotels €360, activities €80. Total €560."}, // corrected
	}}
	res, err := WriteItinerary(context.Background(), llm, brief(), 1)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if res.Steps != 2 || llm.Calls != 2 {
		t.Fatalf("expected a regen (2 steps), got steps=%d calls=%d", res.Steps, llm.Calls)
	}
	if len(res.Violations) != 0 {
		t.Fatalf("corrected text should be clean, got %+v", res.Violations)
	}
}

func TestWriteItinerary_NoRegenSurfacesViolations(t *testing.T) {
	llm := &MockLLM{Script: []Completion{
		{Content: "Fly to Tbilisi for €999."},
	}}
	res, err := WriteItinerary(context.Background(), llm, brief(), 0)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if len(res.Violations) == 0 {
		t.Fatal("with no regen budget the ungrounded price must be surfaced as a violation")
	}
	if res.Violations[0].Value != 999 {
		t.Fatalf("unexpected violation value: %+v", res.Violations)
	}
}
