package pricefetch

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/KhanbalaRashidov/ai-travel-platform/services/ai"
)

// Extracted is one priced option the model pulled out of a fetched page. The
// detail fields are optional and must come verbatim from the page text.
type Extracted struct {
	Price    float64 `json:"price"`
	Currency string  `json:"currency"`
	Operator string  `json:"operator,omitempty"`
	Depart   string  `json:"depart,omitempty"`
	Duration string  `json:"duration,omitempty"`
	Stops    string  `json:"stops,omitempty"`
	Baggage  string  `json:"baggage,omitempty"`
}

type extractEnvelope struct {
	Prices []Extracted `json:"prices"`
}

const extractSystem = "You extract transport fares from fetched web-page text. " +
	"Return STRICT JSON: {\"prices\":[{\"price\":<number>,\"currency\":\"<ISO 4217>\"," +
	"\"operator\":\"\",\"depart\":\"\",\"duration\":\"\",\"stops\":\"\",\"baggage\":\"\"}]}. " +
	"Include ONLY fares that appear literally in the text for traveling the given route. " +
	"For each fare, also copy — ONLY if the text states it for that fare — the operator/airline, " +
	"departure time, total duration, number of stops, and baggage allowance (verbatim). " +
	"Leave a detail field empty when the text does not state it. " +
	"Never invent, estimate, convert or round a number. If no fares are present, return {\"prices\":[]}."

// maxChars caps how much page text we hand the model (cost + context limit).
const maxChars = 12000

// Extract asks the LLM (JSON mode) to pull prices out of raw page text. The
// returned prices are candidates only — they must still pass the verbatim
// guardrail before being trusted.
func Extract(ctx context.Context, llm ai.LLM, raw string, q LegQuery) ([]Extracted, error) {
	if llm == nil || strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	text := raw
	if len(text) > maxChars {
		text = text[:maxChars]
	}
	user := fmt.Sprintf("Route: %s -> %s (%s). Page text:\n%s", q.From, q.To, q.Mode, text)
	comp, err := ai.CompleteStructured(ctx, llm, []ai.Message{
		{Role: ai.RoleSystem, Content: extractSystem},
		{Role: ai.RoleUser, Content: user},
	})
	if err != nil {
		return nil, err
	}
	var env extractEnvelope
	if err := json.Unmarshal([]byte(ai.ExtractJSONObject(comp.Content)), &env); err != nil {
		return nil, fmt.Errorf("parse extracted prices: %w", err)
	}
	return env.Prices, nil
}

// digits keeps only 0-9 so "1 234,50" and "1,234.50" compare by their digits.
var nonDigit = regexp.MustCompile(`[^0-9]`)

func digitsOnly(s string) string { return nonDigit.ReplaceAllString(s, "") }

// Verbatim is the anti-hallucination guardrail: an extracted amount is trusted
// only if its digits appear in the fetched text (as an integer or with 2
// decimals, tolerant of thousands/decimal separators). Pure and unit-tested.
func Verbatim(price float64, raw string) bool {
	if price <= 0 {
		return false
	}
	rawDigits := digitsOnly(raw)
	forms := []string{
		fmt.Sprintf("%.0f", price),
		fmt.Sprintf("%.2f", price),
	}
	for _, f := range forms {
		if d := digitsOnly(f); d != "" && strings.Contains(rawDigits, d) {
			return true
		}
	}
	return false
}
