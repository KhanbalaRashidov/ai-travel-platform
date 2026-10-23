package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// TripEdit is a structured change the LLM extracts from a conversational edit
// request ("make it cheaper", "drop Istanbul", "one more night"). The LLM only
// interprets intent into this shape — it never computes routes or prices, so the
// deterministic engine stays authoritative and the price guardrail holds.
type TripEdit struct {
	AddCities    []string `json:"add_cities"`
	RemoveCities []string `json:"remove_cities"`
	SetTier      string   `json:"set_tier"`     // "" | budget | standard | vip
	SetMode      string   `json:"set_mode"`     // "" | cheapest | flight | train | bus
	SetNights    int      `json:"set_nights"`   // 0 = unset
	NightsDelta  int      `json:"nights_delta"` // e.g. +1, -1
	Cheaper      bool     `json:"cheaper"`      // "make it cheaper"
}

const editSystemPrompt = `You convert a traveler's edit request into a STRICT JSON object with keys: ` +
	`add_cities (array of city names to add), remove_cities (array to remove), ` +
	`set_tier (budget|standard|vip or ""), set_mode (cheapest|flight|train|bus or ""), ` +
	`set_nights (int, 0 if unchanged), nights_delta (int, e.g. 1 or -1), cheaper (bool). ` +
	`Map "cheaper"/"save money" to cheaper=true. Map "add a day"/"one more night" to nights_delta=1. ` +
	`Use only city names the user mentions. Output ONLY the JSON object, no prose.`

// ExtractEdit asks the LLM to turn a free-text edit into a TripEdit and parses
// it. It prefers the provider's JSON mode and tolerates wrapped output; on a
// parse failure it returns an error carrying a short snippet of the raw reply
// so the caller can log what the model actually said. Pure aside from the
// injected LLM, so it is tested with MockLLM.
func ExtractEdit(ctx context.Context, llm LLM, message, currentSummary string) (TripEdit, error) {
	comp, err := completeStructured(ctx, llm, []Message{
		{Role: RoleSystem, Content: editSystemPrompt},
		{Role: RoleUser, Content: "Current plan: " + currentSummary + "\nEdit request: " + message},
	})
	if err != nil {
		return TripEdit{}, err
	}
	var e TripEdit
	if err := json.Unmarshal([]byte(extractJSONObject(comp.Content)), &e); err != nil {
		return TripEdit{}, fmt.Errorf("parse edit: %w (raw: %q)", err, snippet(comp.Content, 200))
	}
	return e, nil
}

// EditState is the subset of wizard parameters a TripEdit can change. City
// matching is by name, case-insensitive; the caller maps names back to its own
// identifiers.
type EditState struct {
	Cities []string `json:"cities"`
	Start  string   `json:"start"`
	Tier   string   `json:"tier"`
	Mode   string   `json:"mode"`
	Nights int      `json:"nights"`
}

var tierOrder = []string{"budget", "standard", "vip"}

func validTier(t string) bool { return t == "budget" || t == "standard" || t == "vip" }
func validMode(m string) bool {
	return m == "cheapest" || m == "flight" || m == "train" || m == "bus"
}

// downgradeTier lowers one comfort step (vip->standard->budget->budget).
func downgradeTier(t string) string {
	for i, v := range tierOrder {
		if v == t && i > 0 {
			return tierOrder[i-1]
		}
	}
	return "budget"
}

func clampNights(n int) int {
	if n < 1 {
		return 1
	}
	if n > 10 {
		return 10
	}
	return n
}

func eqFold(a, b string) bool { return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b)) }

func contains(list []string, v string) bool {
	for _, x := range list {
		if eqFold(x, v) {
			return true
		}
	}
	return false
}

// ApplyEdit applies a TripEdit to the current state deterministically and
// returns the new state plus a list of machine-readable change codes
// (e.g. "mode:cheapest", "removed:Istanbul", "nights:3"). Pure and unit-tested.
func ApplyEdit(s EditState, e TripEdit) (EditState, []string) {
	out := EditState{
		Cities: append([]string(nil), s.Cities...),
		Start:  s.Start,
		Tier:   s.Tier,
		Mode:   s.Mode,
		Nights: s.Nights,
	}
	var changed []string

	for _, rc := range e.RemoveCities {
		if !contains(out.Cities, rc) {
			continue
		}
		kept := out.Cities[:0:0]
		for _, c := range out.Cities {
			if !eqFold(c, rc) {
				kept = append(kept, c)
			}
		}
		out.Cities = kept
		changed = append(changed, "removed:"+rc)
		if eqFold(out.Start, rc) {
			out.Start = ""
		}
	}
	for _, ac := range e.AddCities {
		if ac == "" || contains(out.Cities, ac) {
			continue
		}
		out.Cities = append(out.Cities, ac)
		changed = append(changed, "added:"+ac)
	}
	// Ensure a valid start.
	if out.Start == "" && len(out.Cities) > 0 {
		out.Start = out.Cities[0]
	}

	if e.Cheaper {
		if out.Mode != "cheapest" {
			out.Mode = "cheapest"
			changed = append(changed, "mode:cheapest")
		}
		if d := downgradeTier(out.Tier); d != out.Tier {
			out.Tier = d
			changed = append(changed, "tier:"+d)
		}
	}
	if validTier(e.SetTier) && e.SetTier != out.Tier {
		out.Tier = e.SetTier
		changed = append(changed, "tier:"+e.SetTier)
	}
	if validMode(e.SetMode) && e.SetMode != out.Mode {
		out.Mode = e.SetMode
		changed = append(changed, "mode:"+e.SetMode)
	}
	if e.SetNights > 0 {
		n := clampNights(e.SetNights)
		if n != out.Nights {
			out.Nights = n
			changed = append(changed, fmt.Sprintf("nights:%d", n))
		}
	} else if e.NightsDelta != 0 {
		n := clampNights(out.Nights + e.NightsDelta)
		if n != out.Nights {
			out.Nights = n
			changed = append(changed, fmt.Sprintf("nights:%d", n))
		}
	}
	return out, changed
}
