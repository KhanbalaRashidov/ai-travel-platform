package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// destinationsMaxTokens caps the suggestion JSON (a few countries × several
// cities with short reasons).
const destinationsMaxTokens = 2048

// SuggestedCity is one recommended city for a country (D12). Reason + best season
// are model advice. Rating/review fields are populated ONLY from a real source;
// since none is wired they are forced empty here (no invented stars).
type SuggestedCity struct {
	City          string  `json:"city"`
	Country       string  `json:"country"`
	Reason        string  `json:"reason,omitempty"`
	BestSeason    string  `json:"best_season,omitempty"`
	Rating        float64 `json:"rating,omitempty"`
	ReviewCount   int     `json:"review_count,omitempty"`
	ReviewSummary string  `json:"review_summary,omitempty"`
}

// CountrySuggestion groups a country's recommended cities, most recommended first.
type CountrySuggestion struct {
	Country string          `json:"country"`
	Cities  []SuggestedCity `json:"cities"`
}

// DestinationSuggestions is the AI country-mode result: ranked cities per country
// plus optional budget guidance.
type DestinationSuggestions struct {
	Countries    []CountrySuggestion `json:"countries"`
	BudgetAdvice string              `json:"budget_advice,omitempty"`
}

const destinationsSystem = "You are a budget, travel-loving advisor. Given one or more COUNTRIES, list the cities " +
	"tourists most recommend visiting in each, ORDERED by popularity / how strongly they are recommended (most " +
	"recommended first). For EACH city give: a short reason (what it is famous for) and a best_season. " +
	"Give 5-8 cities per country. " +
	"GUARDRAIL: do NOT output star ratings or review counts — set rating to 0 and leave review fields empty (you have " +
	"no real review source). Do NOT invent specific transport or hotel prices. " +
	"When a budget is given, add budget_advice: practical guidance in the stated currency — roughly how many cities and " +
	"days are realistic, which transport mode and comfort tier are economical, and a sensible city combination that " +
	"fits — as general advice, with NO fabricated exact prices. " +
	"Return STRICT JSON: {\"countries\":[{\"country\":\"\",\"cities\":[{\"city\":\"\",\"country\":\"\"," +
	"\"reason\":\"\",\"best_season\":\"\"}]}],\"budget_advice\":\"\"}."

// SuggestCountryDestinations asks the LLM for ranked city suggestions per country
// (D12 country mode), in the given language, with optional budget guidance.
// Ratings/reviews are stripped server-side (guardrail: only real-source ratings,
// never invented). Distinct from SuggestDestinations (minimal budget mode).
func SuggestCountryDestinations(ctx context.Context, llm LLM, countries []string, budgetEUR float64, currency, language string) (DestinationSuggestions, error) {
	if llm == nil {
		return DestinationSuggestions{}, fmt.Errorf("destinations: no llm configured")
	}
	clean := make([]string, 0, len(countries))
	for _, c := range countries {
		if s := strings.TrimSpace(c); s != "" {
			clean = append(clean, s)
		}
	}
	if len(clean) == 0 {
		return DestinationSuggestions{}, fmt.Errorf("destinations: no countries given")
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Countries: %s\n", strings.Join(clean, ", "))
	if budgetEUR > 0 {
		cur := strings.TrimSpace(currency)
		if cur == "" {
			cur = "EUR"
		}
		fmt.Fprintf(&sb, "Total budget: about %.0f %s (give budget_advice for this).\n", budgetEUR, cur)
	}
	if lang := strings.TrimSpace(language); lang != "" {
		fmt.Fprintf(&sb, "Write all text in %s (keep city and country names recognizable).", lang)
	}

	comp, err := completeStructured(WithMaxTokens(ctx, destinationsMaxTokens), llm, []Message{
		{Role: RoleSystem, Content: destinationsSystem},
		{Role: RoleUser, Content: sb.String()},
	})
	if err != nil {
		return DestinationSuggestions{}, err
	}
	var parsed DestinationSuggestions
	if err := json.Unmarshal([]byte(extractJSONObject(comp.Content)), &parsed); err != nil {
		return DestinationSuggestions{}, fmt.Errorf("destinations: parse: %w (raw: %q)", err, snippet(comp.Content, 160))
	}

	// Guardrail: never surface ratings/reviews (no real source wired) — strip any
	// the model may have produced so the UI can't show fabricated stars.
	for i := range parsed.Countries {
		for j := range parsed.Countries[i].Cities {
			parsed.Countries[i].Cities[j].Rating = 0
			parsed.Countries[i].Cities[j].ReviewCount = 0
			parsed.Countries[i].Cities[j].ReviewSummary = ""
			if parsed.Countries[i].Cities[j].Country == "" {
				parsed.Countries[i].Cities[j].Country = parsed.Countries[i].Country
			}
		}
	}
	return parsed, nil
}
