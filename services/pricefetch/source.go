package pricefetch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	conn "github.com/KhanbalaRashidov/ai-travel-platform/services/connectors"
)

// realisticUA is a normal desktop browser UA. We identify as a browser because
// we fetch the same public pages a user would; we do NOT rotate identities or
// bypass challenges — a source that blocks us is skipped, never defeated.
const realisticUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0 Safari/537.36"

// HTTPDoer is the subset of *http.Client used here, so tests inject a fake.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// PriceSource fetches a public page likely to contain a price for a leg.
type PriceSource interface {
	Name() string
	// ToS returns a short, honest note on the source's terms/scraping risk.
	ToS() string
	// Fetch returns the page text + its URL, or ErrUnavailable to be skipped.
	Fetch(ctx context.Context, q LegQuery) (raw string, url string, err error)
}

// HTTPSource is a generic, configurable source: a URL builder + a realistic UA.
// Anti-bot responses (401/403/429) and other non-2xx are treated as
// ErrUnavailable so the resolver skips the source gracefully.
type HTTPSource struct {
	name  string
	tos   string
	build func(q LegQuery) string
	doer  HTTPDoer
	limit int // max bytes of body to keep (price pages are large)
}

// NewHTTPSource builds a generic source. Pass nil doer for a default client.
func NewHTTPSource(name, tos string, build func(q LegQuery) string, doer HTTPDoer) *HTTPSource {
	if doer == nil {
		doer = &http.Client{Timeout: 20 * time.Second}
	}
	return &HTTPSource{name: name, tos: tos, build: build, doer: doer, limit: 400_000}
}

// Name identifies the source.
func (s *HTTPSource) Name() string { return s.name }

// ToS returns the source's terms note.
func (s *HTTPSource) ToS() string { return s.tos }

// Fetch GETs the built URL with a realistic UA.
func (s *HTTPSource) Fetch(ctx context.Context, q LegQuery) (string, string, error) {
	u := s.build(q)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", u, err
	}
	req.Header.Set("User-Agent", realisticUA)
	req.Header.Set("Accept", "text/html,application/json")

	resp, err := s.doer.Do(req)
	if err != nil {
		return "", u, fmt.Errorf("%w: %v", conn.ErrUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()
	// Anti-bot / rate-limit / error -> skip this source, do not retry-hammer.
	if resp.StatusCode == 401 || resp.StatusCode == 403 || resp.StatusCode == 429 || resp.StatusCode >= 500 {
		return "", u, fmt.Errorf("%w: status %d (likely blocked)", conn.ErrUnavailable, resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return "", u, fmt.Errorf("%w: status %d", conn.ErrUnavailable, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(s.limit)))
	if err != nil {
		return "", u, err
	}
	return string(body), u, nil
}

var _ PriceSource = (*HTTPSource)(nil)
