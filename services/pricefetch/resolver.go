package pricefetch

import (
	"context"
	"encoding/json"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/cache"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/ai"
	conn "github.com/KhanbalaRashidov/ai-travel-platform/services/connectors"
)

// maxConcurrentFetches bounds how many sources we hit in parallel per leg.
const maxConcurrentFetches = 6

// Warm is the durable (DB) fare cache tier: a date-keyed store that survives a
// Redis flush. Optional — the resolver works hot-only without it.
type Warm interface {
	GetFare(ctx context.Context, tenantID, key string) (Candidate, bool)
	PutFare(ctx context.Context, tenantID, key string, c Candidate, ttl time.Duration)
}

// Resolver turns the configured sources into a single cheapest, provenance-
// tagged candidate for a leg. Lookup order: hot (Redis) -> warm (DB) -> cold
// (parallel source fetch + AI extraction + verbatim guardrail). Results are
// written back to warm + hot. With no sources or no LLM it is disabled and
// callers fall back to the labeled estimate.
type Resolver struct {
	sources []PriceSource
	llm     ai.LLM
	conv    Converter
	kv      cache.KV
	warm    Warm
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

// WithWarm attaches a durable (DB) warm cache tier.
func (r *Resolver) WithWarm(w Warm) *Resolver { r.warm = w; return r }

// Enabled reports whether real fetching can run (sources + an LLM present).
func (r *Resolver) Enabled() bool { return r != nil && len(r.sources) > 0 && r.llm != nil }

// Sources lists the configured source names + their ToS notes.
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

// Quote returns the cheapest real candidate for the leg plus the full provenance
// chain (every source that was checked, cheapest first), or ok=false when
// nothing could be fetched/verified. On a hot/warm cache hit the chain is just
// the cached chosen candidate.
func (r *Resolver) Quote(ctx context.Context, q LegQuery, display, tenantID string) (Candidate, []Candidate, bool) {
	if !r.Enabled() {
		return Candidate{}, nil, false
	}
	key := cacheKey(q, display)

	// 1) Hot (Redis).
	if r.kv != nil {
		if raw, err := r.kv.Get(ctx, key); err == nil && raw != "" {
			var c Candidate
			if json.Unmarshal([]byte(raw), &c) == nil && c.Price.Amount > 0 {
				return c, []Candidate{c}, true
			}
		}
	}
	// 2) Warm (DB).
	if r.warm != nil {
		if c, ok := r.warm.GetFare(ctx, tenantID, key); ok && c.Price.Amount > 0 {
			r.setHot(ctx, key, c)
			return c, []Candidate{c}, true
		}
	}

	// 3) Cold: parallel source fetch + extract + guardrail + compare.
	cands := r.gather(ctx, q)
	best, ok := cheapest(cands, display, r.conv)
	if !ok {
		return Candidate{}, nil, false
	}
	sortByDisplay(cands, display, r.conv)
	if r.warm != nil {
		r.warm.PutFare(ctx, tenantID, key, best, r.ttl)
	}
	r.setHot(ctx, key, best)
	return best, cands, true
}

// sortByDisplay orders candidates cheapest-first in the display currency.
func sortByDisplay(cands []Candidate, display string, conv Converter) {
	val := func(c Candidate) float64 {
		v, err := conv.Convert(c.Price.Amount, c.Price.Currency, display)
		if err != nil {
			return math.MaxFloat64
		}
		return v
	}
	sort.Slice(cands, func(i, j int) bool { return val(cands[i]) < val(cands[j]) })
}

func (r *Resolver) setHot(ctx context.Context, key string, c Candidate) {
	if r.kv == nil {
		return
	}
	if b, err := json.Marshal(c); err == nil {
		_ = r.kv.Set(ctx, key, string(b), r.ttl)
	}
}

func appliesTo(s PriceSource, mode conn.TransportMode) bool {
	ms := s.Modes()
	if len(ms) == 0 {
		return true
	}
	for _, m := range ms {
		if m == mode {
			return true
		}
	}
	return false
}

// gather fetches every applicable source in parallel, extracts prices, and keeps
// only those that pass the verbatim guardrail.
func (r *Resolver) gather(ctx context.Context, q LegQuery) []Candidate {
	var (
		mu  sync.Mutex
		out []Candidate
		wg  sync.WaitGroup
		sem = make(chan struct{}, maxConcurrentFetches)
	)
	for _, s := range r.sources {
		if !appliesTo(s, q.Mode) {
			continue
		}
		if ctx.Err() != nil {
			break
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(src PriceSource) {
			defer wg.Done()
			defer func() { <-sem }()
			cands := r.fetchOne(ctx, src, q)
			if len(cands) > 0 {
				mu.Lock()
				out = append(out, cands...)
				mu.Unlock()
			}
		}(s)
	}
	wg.Wait()
	return out
}

// fetchOne fetches + extracts + guards one source.
func (r *Resolver) fetchOne(ctx context.Context, s PriceSource, q LegQuery) []Candidate {
	raw, url, err := s.Fetch(ctx, q)
	if err != nil {
		return nil // graceful skip (blocked/unavailable)
	}
	// Fast path: a regex pre-parser handles currency-marked prices without the
	// LLM. The model is called ONLY when the regex finds nothing — this cuts the
	// vast majority of extraction calls (and latency).
	prices := RegexExtract(raw)
	if len(prices) == 0 {
		prices, err = Extract(ctx, r.llm, raw, q)
		if err != nil {
			return nil
		}
	}
	var out []Candidate
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
			Operator:  strings.TrimSpace(p.Operator),
			Depart:    strings.TrimSpace(p.Depart),
			Duration:  strings.TrimSpace(p.Duration),
			Stops:     strings.TrimSpace(p.Stops),
			Baggage:   strings.TrimSpace(p.Baggage),
		})
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
