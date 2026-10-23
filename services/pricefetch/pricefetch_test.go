package pricefetch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/KhanbalaRashidov/ai-travel-platform/services/ai"
	conn "github.com/KhanbalaRashidov/ai-travel-platform/services/connectors"
)

// extractLLM is a stateless (concurrency-safe) fake that extracts the first
// "<num> <CUR>" from the prompt — mirrors a real extractor and is safe under the
// resolver's parallel fetch.
type extractLLM struct{}

var priceRe = regexp.MustCompile(`([0-9]+(?:\.[0-9]+)?)\s*(EUR|USD|AZN)`)

func (extractLLM) Complete(_ context.Context, msgs []ai.Message, _ []ai.ToolSpec) (ai.Completion, error) {
	text := ""
	for _, m := range msgs {
		if m.Role == ai.RoleUser {
			text = m.Content
		}
	}
	mm := priceRe.FindStringSubmatch(text)
	if mm == nil {
		return ai.Completion{Content: `{"prices":[]}`}, nil
	}
	return ai.Completion{Content: fmt.Sprintf(`{"prices":[{"price":%s,"currency":"%s"}]}`, mm[1], mm[2])}, nil
}

// fixedLLM returns a constant body (used to simulate a hallucinated price).
type fixedLLM struct{ body string }

func (f fixedLLM) Complete(context.Context, []ai.Message, []ai.ToolSpec) (ai.Completion, error) {
	return ai.Completion{Content: f.body}, nil
}

// --- test doubles -----------------------------------------------------------

type stubDoer struct {
	status int
	body   string
}

func (d stubDoer) Do(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: d.status, Body: io.NopCloser(strings.NewReader(d.body)), Header: make(http.Header)}, nil
}

// staticConv is a tiny FX converter (value in EUR numeraire).
type staticConv struct{ r map[string]float64 }

func (c staticConv) Convert(a float64, from, to string) (float64, error) {
	if from == to {
		return a, nil
	}
	rf, ok1 := c.r[from]
	rt, ok2 := c.r[to]
	if !ok1 || !ok2 || rt == 0 {
		return 0, fmt.Errorf("no rate")
	}
	return a * rf / rt, nil
}

func conv() Converter { return staticConv{r: map[string]float64{"EUR": 1, "USD": 0.9, "AZN": 0.53}} }

// --- source ----------------------------------------------------------------

func TestHTTPSource_BlockedIsSkipped(t *testing.T) {
	s := NewHTTPSource("blocked", "test", func(LegQuery) string { return "http://x" }, NewHTTPRenderer(stubDoer{status: 403, body: "nope"}), nil)
	if _, _, err := s.Fetch(context.Background(), LegQuery{}); err == nil {
		t.Fatal("403 must yield ErrUnavailable so the source is skipped")
	}
}

func TestHTTPSource_OKReturnsBody(t *testing.T) {
	long := "Cheapest flight from Baku to Tbilisi today is 120 EUR one way, book direct. " + strings.Repeat("info ", 150)
	s := NewHTTPSource("ok", "test", func(LegQuery) string { return "http://x" }, NewHTTPRenderer(stubDoer{status: 200, body: long}), nil)
	raw, _, err := s.Fetch(context.Background(), LegQuery{})
	if err != nil || !strings.Contains(raw, "120") {
		t.Fatalf("expected body, got raw=%q err=%v", raw, err)
	}
}

// --- guardrail --------------------------------------------------------------

func TestVerbatim(t *testing.T) {
	raw := "Cheapest fare from Baku to Tbilisi is 1,234.50 AZN today"
	if !Verbatim(1234.50, raw) {
		t.Fatal("price present (with separators) should pass")
	}
	if !Verbatim(1234, raw) {
		t.Fatal("integer digits present should pass")
	}
	if Verbatim(999, raw) {
		t.Fatal("absent price must fail (hallucination blocked)")
	}
}

// --- resolver end-to-end ----------------------------------------------------

func fakeSource(name, body string) PriceSource {
	// Pad so the body passes the looksEmpty heuristic (real pages are large).
	padded := body + " " + strings.Repeat("detail ", 120)
	return NewHTTPSource(name, "test-only, public fixture", func(LegQuery) string { return "http://x/" + name }, NewHTTPRenderer(stubDoer{status: 200, body: padded}), nil)
}

func TestResolver_ExtractGuardCompare(t *testing.T) {
	// Source A text has 120 EUR; source B text has 100 USD (~90 EUR). Sources are
	// fetched in parallel, so the extractor must be stateless.
	srcA := fakeSource("A", "Flight Baku-Tbilisi from 120 EUR one way")
	srcB := fakeSource("B", "Baku to Tbilisi tickets 100 USD")

	r := NewResolver([]PriceSource{srcA, srcB}, extractLLM{}, conv(), nil, 0)
	if !r.Enabled() {
		t.Fatal("resolver with sources + llm should be enabled")
	}
	cand, ok := r.Quote(context.Background(), LegQuery{From: "Baku", To: "Tbilisi", Mode: conn.ModeFlight}, "EUR", "t1")
	if !ok {
		t.Fatal("expected a candidate")
	}
	// 100 USD -> 90 EUR vs 120 EUR -> cheapest is source B's 100 USD.
	if cand.Source != "B" || cand.Price.Currency != "USD" || cand.Price.Amount != 100 {
		t.Fatalf("expected cheapest = B 100 USD, got %+v", cand)
	}
}

func TestResolver_DisabledWithoutSources(t *testing.T) {
	r := NewResolver(nil, extractLLM{}, conv(), nil, 0)
	if r.Enabled() {
		t.Fatal("no sources -> disabled")
	}
	if _, ok := r.Quote(context.Background(), LegQuery{}, "EUR", "t1"); ok {
		t.Fatal("disabled resolver must not return a candidate")
	}
}

func TestResolver_AllHallucinatedDropped(t *testing.T) {
	src := fakeSource("A", "No prices on this page at all")
	// The model returns 77 EUR, which is NOT in the fetched text -> guardrail drops it.
	r := NewResolver([]PriceSource{src}, fixedLLM{body: `{"prices":[{"price":77,"currency":"EUR"}]}`}, conv(), nil, 0)
	if _, ok := r.Quote(context.Background(), LegQuery{From: "A", To: "B"}, "EUR", "t1"); ok {
		t.Fatal("a price absent from the text must be dropped -> no candidate")
	}
}

func TestResolver_WarmCacheHit(t *testing.T) {
	src := fakeSource("A", "Baku to Tbilisi 120 EUR")
	warm := NewMemWarm()
	r := NewResolver([]PriceSource{src}, extractLLM{}, conv(), nil, 0).WithWarm(warm)
	q := LegQuery{From: "Baku", To: "Tbilisi", Mode: conn.ModeFlight}
	if _, ok := r.Quote(context.Background(), q, "EUR", "t1"); !ok {
		t.Fatal("expected a candidate (cold)")
	}
	// Second call should be served from warm even if the source now fails.
	if _, ok := warm.GetFare(context.Background(), "t1", cacheKey(q, "EUR")); !ok {
		t.Fatal("cold result must be written to warm")
	}
}
