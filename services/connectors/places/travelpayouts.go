// Package places implements city/airport autocomplete over Travelpayouts'
// public places endpoint — key-less, open, and the same dataset powering
// Aviasales search. It returns provenance-tagged places with coordinates, so
// the planner can price any selected city without a hardcoded list.
package places

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	conn "github.com/KhanbalaRashidov/ai-travel-platform/services/connectors"
)

const (
	sourceName     = "travelpayouts-places"
	defaultBaseURL = "https://autocomplete.travelpayouts.com/places2"
)

// HTTPDoer is the subset of *http.Client used here, so tests inject a fake.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Place is one autocomplete result.
type Place struct {
	Code            string  `json:"code"`    // IATA (city or airport)
	Name            string  `json:"name"`    // display name
	Country         string  `json:"country"` // country name
	Lat             float64 `json:"lat"`
	Lng             float64 `json:"lng"`
	Kind            string  `json:"kind"` // "city" | "airport"
	conn.Provenance `json:"provenance"`
}

// Client queries the Travelpayouts places autocomplete.
type Client struct {
	baseURL string
	http    HTTPDoer
	now     func() time.Time
}

// New builds a places client. Pass nil doer for a default HTTP client.
func New(doer HTTPDoer) *Client {
	if doer == nil {
		doer = &http.Client{Timeout: 10 * time.Second}
	}
	return &Client{baseURL: defaultBaseURL, http: doer, now: time.Now}
}

// Name identifies the source for provenance.
func (c *Client) Name() string { return sourceName }

// Search returns places matching term (cities + airports) in the given locale
// (names localized by the API), newest-first by relevance.
func (c *Client) Search(ctx context.Context, term, locale string, limit int) ([]Place, error) {
	term = strings.TrimSpace(term)
	if term == "" {
		return nil, nil
	}
	if locale == "" {
		locale = "en"
	}
	u := fmt.Sprintf("%s?term=%s&locale=%s&types[]=city&types[]=airport", c.baseURL, url.QueryEscape(term), url.QueryEscape(locale))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", conn.ErrUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", conn.ErrUnavailable, resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return parsePlaces(raw, c.now().UTC(), limit), nil
}

// tpPlace mirrors the Travelpayouts places2 payload.
type tpPlace struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	CountryName string `json:"country_name"`
	Type        string `json:"type"`
	Coordinates struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
	} `json:"coordinates"`
}

// parsePlaces turns the API payload into named, geolocated places. It drops
// entries without a name or coordinates and caps at limit. Pure, so it is
// unit-tested without the network.
func parsePlaces(body []byte, fetchedAt time.Time, limit int) []Place {
	var in []tpPlace
	if err := json.Unmarshal(body, &in); err != nil {
		return nil
	}
	prov := conn.Provenance{Source: sourceName, FetchedAt: fetchedAt}
	out := make([]Place, 0, len(in))
	for _, p := range in {
		if strings.TrimSpace(p.Name) == "" || (p.Coordinates.Lat == 0 && p.Coordinates.Lon == 0) {
			continue
		}
		kind := p.Type
		if kind != "airport" {
			kind = "city"
		}
		out = append(out, Place{
			Code: p.Code, Name: p.Name, Country: p.CountryName,
			Lat: p.Coordinates.Lat, Lng: p.Coordinates.Lon, Kind: kind, Provenance: prov,
		})
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}
