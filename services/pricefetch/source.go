package pricefetch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	conn "github.com/KhanbalaRashidov/ai-travel-platform/services/connectors"
)

// realisticUA is a normal desktop browser UA. We fetch the same public pages a
// user would and identify as a browser; we do NOT rotate identities, solve
// CAPTCHAs, or otherwise defeat bot-detection — a blocked source is skipped.
const realisticUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0 Safari/537.36"

// HTTPDoer is the subset of *http.Client used here, so tests inject a fake.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Renderer fetches a URL and returns the page text. HTTPRenderer does a plain
// GET; a headless (Playwright/Chromium) renderer implements the same interface
// to run JS-rendered pages. Both are injectable, so the pipeline is testable.
type Renderer interface {
	Render(ctx context.Context, url string) (body string, status int, err error)
}

// HTTPRenderer is the default renderer: a single GET with a realistic UA.
type HTTPRenderer struct {
	doer  HTTPDoer
	limit int64
}

// NewHTTPRenderer builds the HTTP renderer (nil doer -> default client).
func NewHTTPRenderer(doer HTTPDoer) *HTTPRenderer {
	if doer == nil {
		doer = &http.Client{Timeout: 20 * time.Second}
	}
	return &HTTPRenderer{doer: doer, limit: 500_000}
}

// Render GETs url with browser-like headers.
func (r *HTTPRenderer) Render(ctx context.Context, url string) (string, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("User-Agent", realisticUA)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/json;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	resp, err := r.doer.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, r.limit))
	if err != nil {
		return "", resp.StatusCode, err
	}
	return string(body), resp.StatusCode, nil
}

// PriceSource fetches a public page likely to contain a price for a leg.
type PriceSource interface {
	Name() string
	ToS() string
	Fetch(ctx context.Context, q LegQuery) (raw string, url string, err error)
}

// HTTPSource is a configurable source: a URL builder + an HTTP renderer, with an
// optional headless renderer used when the HTTP body looks empty/JS-only. Anti-
// bot responses (401/403/429/5xx) are treated as ErrUnavailable so the resolver
// skips the source gracefully.
type HTTPSource struct {
	name     string
	tos      string
	build    func(q LegQuery) string
	http     Renderer
	headless Renderer // optional; used when HTTP returns an empty/JS-only page
}

// NewHTTPSource builds a source. Pass nil http renderer for the default; headless
// may be nil (then no JS fallback).
func NewHTTPSource(name, tos string, build func(q LegQuery) string, httpR, headless Renderer) *HTTPSource {
	if httpR == nil {
		httpR = NewHTTPRenderer(nil)
	}
	return &HTTPSource{name: name, tos: tos, build: build, http: httpR, headless: headless}
}

// Name identifies the source.
func (s *HTTPSource) Name() string { return s.name }

// ToS returns the source's terms note.
func (s *HTTPSource) ToS() string { return s.tos }

// blocked reports an anti-bot / error status worth skipping (not retry-hammering).
func blocked(status int) bool {
	return status == 401 || status == 403 || status == 429 || status >= 500
}

// looksEmpty is a heuristic for a JS-rendered shell with no useful text.
func looksEmpty(body string) bool {
	t := strings.TrimSpace(body)
	return len(t) < 600 || (!strings.ContainsAny(t, "0123456789") && len(t) < 4000)
}

// Fetch renders the built URL (HTTP first, headless fallback on empty), skipping
// gracefully on anti-bot/error.
func (s *HTTPSource) Fetch(ctx context.Context, q LegQuery) (string, string, error) {
	u := s.build(q)
	body, status, err := s.http.Render(ctx, u)
	if err != nil {
		return "", u, fmt.Errorf("%w: %v", conn.ErrUnavailable, err)
	}
	if blocked(status) {
		return "", u, fmt.Errorf("%w: status %d (blocked)", conn.ErrUnavailable, status)
	}
	// JS-rendered / empty page -> try the headless renderer if present.
	if s.headless != nil && (status != http.StatusOK || looksEmpty(body)) {
		if hb, hs, herr := s.headless.Render(ctx, u); herr == nil && !blocked(hs) && !looksEmpty(hb) {
			return hb, u, nil
		}
	}
	if status != http.StatusOK {
		return "", u, fmt.Errorf("%w: status %d", conn.ErrUnavailable, status)
	}
	return body, u, nil
}

var _ PriceSource = (*HTTPSource)(nil)
