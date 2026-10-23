package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// TripConstraints is the structured intent extracted from a user's free-text
// request. The LLM produces it as JSON; the planner consumes it.
type TripConstraints struct {
	Cities        []string `json:"cities"`
	SuggestCities bool     `json:"suggest_cities"`
	Days          int      `json:"days"`
	BudgetEUR     float64  `json:"budget_eur"`
	Tier          string   `json:"tier"`           // budget | standard | vip
	TransportPref string   `json:"transport_pref"` // cheapest | flight | train | bus
	Tempo         string   `json:"tempo"`          // dense | relaxed
	Interests     []string `json:"interests"`
	GroupSize     int      `json:"group_size"`
}

const extractSystemPrompt = `Extract trip constraints as STRICT JSON with keys: ` +
	`cities (array of city names), suggest_cities (bool), days (int), budget_eur (number), ` +
	`tier (budget|standard|vip), transport_pref (cheapest|flight|train|bus), ` +
	`tempo (dense|relaxed), interests (array of strings), group_size (int). ` +
	`Output ONLY the JSON object, no prose.`

// ExtractConstraints asks the LLM to turn free text into TripConstraints,
// parses the JSON, and normalizes/validates it with safe defaults. Pure aside
// from the injected LLM, so it is tested with MockLLM (no network).
func ExtractConstraints(ctx context.Context, llm LLM, userText string) (TripConstraints, error) {
	comp, err := completeStructured(ctx, llm, []Message{
		{Role: RoleSystem, Content: extractSystemPrompt},
		{Role: RoleUser, Content: userText},
	})
	if err != nil {
		return TripConstraints{}, err
	}
	var c TripConstraints
	if err := json.Unmarshal([]byte(extractJSONObject(comp.Content)), &c); err != nil {
		return TripConstraints{}, fmt.Errorf("parse constraints: %w (raw: %q)", err, snippet(comp.Content, 200))
	}
	c.normalize()
	return c, nil
}

func (c *TripConstraints) normalize() {
	switch strings.ToLower(strings.TrimSpace(c.Tier)) {
	case "budget":
		c.Tier = "budget"
	case "vip":
		c.Tier = "vip"
	default:
		c.Tier = "standard"
	}
	switch strings.ToLower(strings.TrimSpace(c.TransportPref)) {
	case "flight":
		c.TransportPref = "flight"
	case "train":
		c.TransportPref = "train"
	case "bus":
		c.TransportPref = "bus"
	default:
		c.TransportPref = "cheapest"
	}
	if c.Tempo != "dense" {
		c.Tempo = "relaxed"
	}
	if c.GroupSize < 1 {
		c.GroupSize = 1
	}
	if c.Days < 0 {
		c.Days = 0
	}
	if len(c.Cities) == 0 {
		c.SuggestCities = true
	}
}

// stripFences removes a leading ```lang fence and trailing ``` if present, so a
// fenced JSON reply still parses.
func stripFences(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		if i := strings.IndexByte(s, '\n'); i >= 0 {
			s = s[i+1:]
		}
		s = strings.TrimSuffix(strings.TrimSpace(s), "```")
	}
	return strings.TrimSpace(s)
}
