package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const suggestSystem = "You are a budget-savvy travel planner. Given an origin city, a total budget " +
	"in EUR and a trip length in days, suggest an ordered set of 2–4 affordable, popular destination " +
	"CITIES (real city names, no airports) reachable from the origin that together fit the budget and " +
	"days. Return STRICT JSON {\"cities\":[\"City\",...]} — city names only, no prose. Do not include the origin."

type suggestResp struct {
	Cities []string `json:"cities"`
}

// SuggestDestinations asks the LLM for budget-fitting destination cities for the
// minimal ("I only know where I start + my budget") mode. It returns plain city
// names (the caller geocodes them); prices are never taken from here — the
// deterministic engine prices the resulting plan. Tested with MockLLM.
func SuggestDestinations(ctx context.Context, llm LLM, origin string, budgetEUR float64, days int) ([]string, error) {
	if llm == nil {
		return nil, fmt.Errorf("suggest: no llm configured")
	}
	if days < 1 {
		days = 5
	}
	user := fmt.Sprintf("Origin: %s. Budget: %.0f EUR. Days: %d.", origin, budgetEUR, days)
	comp, err := completeStructured(ctx, llm, []Message{
		{Role: RoleSystem, Content: suggestSystem},
		{Role: RoleUser, Content: user},
	})
	if err != nil {
		return nil, err
	}
	var r suggestResp
	if err := json.Unmarshal([]byte(extractJSONObject(comp.Content)), &r); err != nil {
		return nil, fmt.Errorf("suggest: parse: %w (raw: %q)", err, snippet(comp.Content, 160))
	}
	// Clean: trim, drop empties + the origin, de-dup, cap at 4.
	seen := map[string]bool{}
	out := make([]string, 0, 4)
	for _, c := range r.Cities {
		c = strings.TrimSpace(c)
		if c == "" || strings.EqualFold(c, strings.TrimSpace(origin)) {
			continue
		}
		key := strings.ToLower(c)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, c)
		if len(out) == 4 {
			break
		}
	}
	return out, nil
}
