package ai

import (
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Currency markers we recognize when deciding whether a number is a price.
// Only currency-tagged numbers are treated as prices, so plain counts like
// "3 days" are never mistaken for money.
const currencyAlt = `€|\$|₼|£|AZN|EUR|USD|GBP|TRY|RUB`

var (
	// currency BEFORE the number: "€120", "AZN 1.84"
	preMoneyRe = regexp.MustCompile(`(?i)(?:` + currencyAlt + `)\s?([0-9]+(?:[.,][0-9]+)?)`)
	// currency AFTER the number: "120 EUR", "40€"
	postMoneyRe = regexp.MustCompile(`(?i)([0-9]+(?:[.,][0-9]+)?)\s?(?:` + currencyAlt + `)`)
)

// ExtractMoneyValues returns every currency-tagged amount found in s, rounded to
// 2 decimals. It is pure and used both to harvest grounding facts from tool
// results and to validate the final text.
func ExtractMoneyValues(s string) []float64 {
	var out []float64
	for _, re := range []*regexp.Regexp{preMoneyRe, postMoneyRe} {
		for _, m := range re.FindAllStringSubmatch(s, -1) {
			if v, ok := parseAmount(m[1]); ok {
				out = append(out, v)
			}
		}
	}
	return out
}

func parseAmount(raw string) (float64, bool) {
	// normalize a decimal comma to a dot ("1,84" -> "1.84").
	norm := strings.Replace(raw, ",", ".", 1)
	f, err := strconv.ParseFloat(norm, 64)
	if err != nil {
		return 0, false
	}
	return math.Round(f*100) / 100, true
}
