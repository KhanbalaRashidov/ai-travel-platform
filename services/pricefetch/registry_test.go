package pricefetch

import (
	"context"
	"strings"
	"testing"
)

func TestParseSpecAndBuildURL(t *testing.T) {
	spec := `
# a comment
sky | metasearch, ToS restricts | https://ex.com/{from}/{to}/{date}?adults={adults}
bad-line-without-pipes
kiwi | aggregator | https://kiwi.example/{from}-{to}
`
	srcs := SourcesFromSpec(spec, NewHTTPRenderer(stubDoer{status: 200, body: ""}), nil)
	if len(srcs) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(srcs))
	}
	if srcs[0].Name() != "sky" || !strings.Contains(srcs[0].ToS(), "ToS") {
		t.Fatalf("source 0 wrong: %s / %s", srcs[0].Name(), srcs[0].ToS())
	}
	u := buildURL("https://ex.com/{from}/{to}/{date}?adults={adults}", LegQuery{From: "Baku", To: "Tbilisi", DepartDate: "2026-07-01", Adults: 2})
	if !strings.Contains(u, "Baku") || !strings.Contains(u, "Tbilisi") || !strings.Contains(u, "2026-07-01") || !strings.Contains(u, "adults=2") {
		t.Fatalf("buildURL substitution wrong: %s", u)
	}
}

type fakeRenderer struct {
	body   string
	status int
}

func (f fakeRenderer) Render(context.Context, string) (string, int, error) {
	return f.body, f.status, nil
}

func TestHeadlessFallbackOnEmptyPage(t *testing.T) {
	// HTTP returns a tiny JS shell (looksEmpty); headless returns the real page.
	httpR := NewHTTPRenderer(stubDoer{status: 200, body: "<html><body></body></html>"})
	rich := "Baku to Tbilisi flight 135 EUR " + strings.Repeat("x ", 400)
	headless := fakeRenderer{body: rich, status: 200}
	s := NewHTTPSource("js-site", "test", func(LegQuery) string { return "http://x" }, httpR, headless)

	raw, _, err := s.Fetch(context.Background(), LegQuery{From: "Baku", To: "Tbilisi"})
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if !strings.Contains(raw, "135 EUR") {
		t.Fatalf("expected headless-rendered body, got %q", raw[:min(60, len(raw))])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
