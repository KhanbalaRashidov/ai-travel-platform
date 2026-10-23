// Package fx implements FX-rate connectors. Frankfurter (ECB) is a free,
// key-less, open-data source — ideal for the "never depend on a gated API"
// rule. Rates are daily; good enough for indicative cost estimates.
package fx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/KhanbalaRashidov/ai-travel-platform/services/connectors"
)

// defaultBaseURL is the Frankfurter ECB API root.
const defaultBaseURL = "https://api.frankfurter.dev/v1"

// HTTPDoer is the subset of *http.Client used here, so tests can inject a fake.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Frankfurter is an FxSource backed by the Frankfurter ECB API.
type Frankfurter struct {
	baseURL string
	http    HTTPDoer
	now     func() time.Time
}

// New builds a Frankfurter connector. Pass nil to use a default HTTP client.
func New(doer HTTPDoer) *Frankfurter {
	if doer == nil {
		doer = &http.Client{Timeout: 10 * time.Second}
	}
	return &Frankfurter{baseURL: defaultBaseURL, http: doer, now: time.Now}
}

// Name identifies the source for provenance.
func (f *Frankfurter) Name() string { return "frankfurter-ecb" }

// Rate fetches base->quote. Same currency returns 1.0 without a network call.
func (f *Frankfurter) Rate(ctx context.Context, base, quote string) (connectors.FxRate, error) {
	base, quote = strings.ToUpper(base), strings.ToUpper(quote)
	fetchedAt := f.now().UTC()
	if base == quote {
		return connectors.FxRate{Base: base, Quote: quote, Rate: 1,
			Provenance: connectors.Provenance{Source: f.Name(), FetchedAt: fetchedAt, ExpiresAt: endOfDay(fetchedAt)}}, nil
	}

	u := fmt.Sprintf("%s/latest?base=%s&symbols=%s", f.baseURL, url.QueryEscape(base), url.QueryEscape(quote))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return connectors.FxRate{}, err
	}
	resp, err := f.http.Do(req)
	if err != nil {
		return connectors.FxRate{}, fmt.Errorf("%w: %v", connectors.ErrUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return connectors.FxRate{}, fmt.Errorf("%w: status %d", connectors.ErrUnavailable, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return connectors.FxRate{}, err
	}
	return parseRate(body, base, quote, fetchedAt, f.Name())
}

// frankfurterResp mirrors the API payload.
type frankfurterResp struct {
	Base  string             `json:"base"`
	Date  string             `json:"date"`
	Rates map[string]float64 `json:"rates"`
}

// parseRate extracts the quote rate from a Frankfurter response. Pure, so it is
// unit-tested without the network.
func parseRate(body []byte, base, quote string, fetchedAt time.Time, source string) (connectors.FxRate, error) {
	var r frankfurterResp
	if err := json.Unmarshal(body, &r); err != nil {
		return connectors.FxRate{}, fmt.Errorf("decode frankfurter response: %w", err)
	}
	rate, ok := r.Rates[quote]
	if !ok || rate <= 0 {
		return connectors.FxRate{}, fmt.Errorf("%w: rate for %s not found", connectors.ErrUnavailable, quote)
	}
	return connectors.FxRate{
		Base:  base,
		Quote: quote,
		Rate:  rate,
		Provenance: connectors.Provenance{
			Source:    source,
			FetchedAt: fetchedAt,
			ExpiresAt: endOfDay(fetchedAt),
		},
	}, nil
}

// endOfDay is the natural expiry for daily ECB rates.
func endOfDay(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 23, 59, 59, 0, time.UTC)
}

var _ connectors.FxSource = (*Frankfurter)(nil)
