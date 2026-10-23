package pricefetch

import (
	"net/url"
	"strings"
)

// SourceSpec is one line of the PRICE_SOURCES config:
//
//	name | tos-note | url-template
//
// The template may contain {from} {to} {date} {adults}; {from}/{to} are URL-
// query-escaped. Lines starting with '#' and blank lines are ignored.
//
// Example (one per line):
//
//	skyscanner | metasearch, ToS restricts scraping; user-accepted | https://www.skyscanner.net/transport/flights/{from}/{to}/{date}/
//	kiwi       | aggregator, public search | https://www.kiwi.com/en/search/results/{from}/{to}/{date}
//
// Shipping no built-in targets keeps the code honest about ToS; the operator
// decides which sites to scrape via .env (PRICE_SOURCES) and accepts the terms.
func parseSpec(spec string) []struct{ name, tos, tmpl string } {
	var out []struct{ name, tos, tmpl string }
	for _, line := range strings.Split(spec, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		if len(parts) != 3 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		tmpl := strings.TrimSpace(parts[2])
		if name == "" || tmpl == "" {
			continue
		}
		out = append(out, struct{ name, tos, tmpl string }{name, strings.TrimSpace(parts[1]), tmpl})
	}
	return out
}

// buildURL substitutes a leg into a template.
func buildURL(tmpl string, q LegQuery) string {
	rep := strings.NewReplacer(
		"{from}", url.QueryEscape(firstNonEmpty(q.From)),
		"{to}", url.QueryEscape(firstNonEmpty(q.To)),
		"{date}", url.QueryEscape(q.DepartDate),
		"{adults}", itoa(q.Adults),
	)
	return rep.Replace(tmpl)
}

func firstNonEmpty(s string) string { return strings.TrimSpace(s) }

func itoa(n int) string {
	if n <= 0 {
		return "1"
	}
	// small positive ints only
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}

// SourcesFromSpec builds PriceSources from a PRICE_SOURCES config string. Each
// source fetches via the HTTP renderer, falling back to headless (if provided)
// on empty/JS pages.
func SourcesFromSpec(spec string, httpR, headless Renderer) []PriceSource {
	var sources []PriceSource
	for _, s := range parseSpec(spec) {
		tmpl := s.tmpl
		sources = append(sources, NewHTTPSource(s.name, s.tos, func(q LegQuery) string {
			return buildURL(tmpl, q)
		}, httpR, headless))
	}
	return sources
}
