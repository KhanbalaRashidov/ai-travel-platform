package pricefetch

import (
	"context"
	"encoding/json"
	"math"
	"strings"
	"time"

	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/cache"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/ai"
	conn "github.com/KhanbalaRashidov/ai-travel-platform/services/connectors"
)

// Resolver turns the configured sources into a single cheapest, provenance-
// tagged candidate for a leg, with a hot-cache (so repeat queries skip the
// network). With no sources or no LLM it is disabled and callers fall back to
// the labeled estimate.
type Resolver struct {
	sources []PriceSource
	llm     ai.LLM
	conv    Converter
	kv      cache.KV
	ttl     time.Duration
	now     func() time.Time
}

// NewResolver wires the resolver. ttl <=0 defaults to 6h.
func NewResolver(sources []PriceSource, llm ai.LLM, conv Converter, kv cache.KV, ttl time.Duration) *Resolver {
	if ttl <= 0 {
		ttl = 6 * time.Hour
	}
	return &Resolver{sources: sources, llm: llm, conv: conv, kv: kv, ttl: ttl, now: time.Now}
}

// Enabled reports whether real fetching can run (sources + an LLM present).
func (r *Resolver) Enabled() bool { return r != nil && len(r.sources) > 0 && r.llm != nil }

// Sources lists the configured source names + their ToS notes (for /health-style
// introspection and honesty in the UI).
func (r *Resolver) Sources() map[string]string {
	out := make(map[string]string, len(r.sources))
	for _, s := range r.sources {
		out[s.Name()] = s.ToS()
	}
	return out
}

func cacheKey(q LegQuery, display string) string {
	return strings.Join([]string{"fare", q.From, q.To, string(q.Mode), q.DepartDate, display}, "|")
}

// Quote returns the cheapest real candidate for the leg in `display` currency,
// or ok=false when nothing could be fetched/verified.
func (r *Resolver) Quote(ctx context.Context, q LegQuery, display string) (Candidate, bool) {
	if !r.Enabled() {
		return Candidate{}, false
	}
	key := cacheKey(q, display)
	if r.kv != nil {
		if raw, err := r.kv.Get(ctx, key); err == nil && raw != "" {
			var c Candidate
			if json.Unmarshal([]byte(raw), &c) == nil && c.Price.Amount > 0 {
				return c, true
			}
		}
	}

	candidates := r.gather(ctx, q)
	best, ok := cheapest(candidates, display, r.conv)
	if !ok {
		return Candidate{}, false
	}
	if r.kv != nil {
		if b, err := json.Marshal(best); err == nil {
			_ = r.kv.Set(ctx, key, string(b), r.ttl)
		}
	}
	return best, true
}

// gather fetches every source, extracts prices, and keeps only those that pass
// the verbatim guardrail.
func (r *Resolver) gather(ctx context.Context, q LegQuery) []Candidate {
	var out []Candidate
	for _, s := range r.sources {
		if ctx.Err() != nil {
			break
		}
		raw, url, err := s.Fetch(ctx, q)
		if err != nil {
			continue // graceful skip (blocked/unavailable)
		}
		prices, err := Extract(ctx, r.llm, raw, q)
		if err != nil {
			continue
		}
		for _, p := range prices {
			if p.Price <= 0 || strings.TrimSpace(p.Currency) == "" {
				continue
			}
			if !Verbatim(p.Price, raw) {
				continue // GUARDRAIL: amount not present verbatim -> drop (no hallucination)
			}
			out = append(out, Candidate{
				Source:    s.Name(),
				URL:       url,
				Price:     conn.Money{Amount: p.Price, Currency: strings.ToUpper(strings.TrimSpace(p.Currency))},
				FetchedAt: r.now().UTC(),
			})
		}
	}
	return out
}

// cheapest picks the lowest candidate after normalizing to the display currency.
func cheapest(cands []Candidate, display string, conv Converter) (Candidate, bool) {
	best := Candidate{}
	bestVal := math.MaxFloat64
	found := false
	for _, c := range cands {
		v, err := conv.Convert(c.Price.Amount, c.Price.Currency, display)
		if err != nil {
			continue
		}
		if v < bestVal {
			bestVal, best, found = v, c, true
		}
	}
	return best, found
}
