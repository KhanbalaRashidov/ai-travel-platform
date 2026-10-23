package ai

import (
	"context"
	"fmt"
	"strings"
)

// LegFact is one priced transport leg the writer may describe.
type LegFact struct {
	From  string
	To    string
	Mode  string
	Price float64
}

// PlanBrief is the structured, already-priced plan the itinerary writer turns
// into prose. The writer must use ONLY these numbers — it never prices anything
// itself — which is what keeps the price-hallucination guardrail meaningful.
type PlanBrief struct {
	Cities          []string
	Legs            []LegFact
	Nights          int
	Currency        string
	TransportTotal  float64
	HotelsTotal     float64
	ActivitiesTotal float64
	Total           float64
}

// grounding returns every price the writer is allowed to mention.
func (b PlanBrief) grounding() []float64 {
	g := make([]float64, 0, len(b.Legs)+4)
	for _, l := range b.Legs {
		g = append(g, l.Price)
	}
	g = append(g, b.TransportTotal, b.HotelsTotal, b.ActivitiesTotal, b.Total)
	return g
}

// factSheet renders the brief as an explicit, deterministic fact list fed to the
// model as the single source of truth.
func (b PlanBrief) factSheet() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Route: %s\n", strings.Join(b.Cities, " -> "))
	fmt.Fprintf(&sb, "Nights per city: %d\n", b.Nights)
	fmt.Fprintf(&sb, "Currency: %s\n", b.Currency)
	sb.WriteString("Legs:\n")
	for _, l := range b.Legs {
		fmt.Fprintf(&sb, "  - %s to %s by %s: %.2f %s\n", l.From, l.To, l.Mode, l.Price, b.Currency)
	}
	fmt.Fprintf(&sb, "Transport total: %.2f %s\n", b.TransportTotal, b.Currency)
	fmt.Fprintf(&sb, "Hotels total: %.2f %s\n", b.HotelsTotal, b.Currency)
	fmt.Fprintf(&sb, "Activities total: %.2f %s\n", b.ActivitiesTotal, b.Currency)
	fmt.Fprintf(&sb, "Grand total: %.2f %s\n", b.Total, b.Currency)
	return sb.String()
}

const writerSystem = "You are a concise, friendly travel-itinerary writer for an advisory-only " +
	"planner (no booking, no payment). Write a short day-by-day plan from the facts provided. " +
	"Use ONLY the prices in the facts; never invent, estimate or round prices. Prices are " +
	"indicative. Keep it practical and warm. Do not promise availability."

// WriteItinerary asks the model to turn a priced PlanBrief into a day-by-day
// narrative, then enforces the price guardrail: any price not grounded in the
// brief triggers a rewrite (up to maxRegen). If it still fails, the text is
// returned WITH its violations rather than silently trusted — identical policy
// to the orchestrator.
func WriteItinerary(ctx context.Context, llm LLM, b PlanBrief, maxRegen int) (Result, error) {
	grounding := b.grounding()
	msgs := []Message{
		{Role: RoleSystem, Content: writerSystem},
		{Role: RoleUser, Content: "Plan facts:\n" + b.factSheet() + "\nWrite the itinerary."},
	}
	regenLeft := maxRegen
	for step := 1; ; step++ {
		comp, err := llm.Complete(ctx, msgs, nil)
		if err != nil {
			return Result{}, err
		}
		violations := Validate(comp.Content, grounding)
		if len(violations) == 0 || regenLeft == 0 {
			return Result{Text: comp.Content, Grounding: grounding, Violations: violations, Steps: step}, nil
		}
		regenLeft--
		msgs = append(msgs,
			Message{Role: RoleAssistant, Content: comp.Content},
			Message{Role: RoleSystem, Content: "A price you used is not in the facts. Rewrite using ONLY the prices listed; do not invent numbers."},
		)
	}
}
