package pricefetch

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/KhanbalaRashidov/ai-travel-platform/services/ai"
)

// Extracted is one price the model pulled out of a fetched page.
type Extracted struct {
	Price    float64 `json:"price"`
	Currency string  `json:"currency"`
}

type extractEnvelope struct {
	Prices []Extracted `json:"prices"`
}

const extractSystem = "You extract transport ticket prices from fetched web-page text. " +
	"Return STRICT JSON: {\"prices\":[{\"price\":<number>,\"currency\":\"<ISO 4217>\"}]}. " +
	"Include ONLY prices that appear literally in the text for traveling the given route. " +
	"Never invent, estimate, convert or round a number. If none are present, return {\"prices\":[]}."

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
