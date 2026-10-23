// Package poi implements a points-of-interest connector over the OpenStreetMap
// Overpass API — free, key-less, open data (ODbL, attribution required). It
// queries tourism/historic features in a small bounding box around a city and
// returns provenance-tagged POIs (Estimated=false: this is real data).
package poi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	conn "github.com/KhanbalaRashidov/ai-travel-platform/services/connectors"
)

const (
	sourceName     = "osm-overpass"
	defaultBaseURL = "https://overpass-api.de/api/interpreter"
	// bboxHalf is half the side of the lookup box in degrees (~11 km at the equator).
	bboxHalf = 0.1
)

// HTTPDoer is the subset of *http.Client used here, so tests inject a fake.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Overpass is a POISource backed by the OSM Overpass API.
type Overpass struct {
	baseURL string
	geo     conn.Geocoder
	http    HTTPDoer
	ttl     time.Duration
	now     func() time.Time
}

// New builds an Overpass connector. Pass nil doer for a default HTTP client;
// ttl <=0 defaults to 30 days (POIs change slowly).
func New(geo conn.Geocoder, doer HTTPDoer, ttl time.Duration) *Overpass {
	if doer == nil {
		doer = &http.Client{Timeout: 20 * time.Second}
	}
	if ttl <= 0 {
		ttl = 30 * 24 * time.Hour
	}
	return &Overpass{baseURL: defaultBaseURL, geo: geo, http: doer, ttl: ttl, now: time.Now}
}

// Name identifies the source for provenance.
func (o *Overpass) Name() string { return sourceName }

// buildQuery returns the Overpass QL for tourism/historic nodes in a bbox.
func buildQuery(c conn.LatLng) string {
	s, w := c.Lat-bboxHalf, c.Lng-bboxHalf
	n, e := c.Lat+bboxHalf, c.Lng+bboxHalf
	bbox := fmt.Sprintf("%f,%f,%f,%f", s, w, n, e)
	return fmt.Sprintf(
		`[out:json][timeout:20];(node["tourism"~"attraction|museum|gallery|viewpoint|artwork"](%s);node["historic"](%s););out body 60;`,
		bbox, bbox,
	)
}

// POIs fetches up to limit points of interest for a city.
func (o *Overpass) POIs(ctx context.Context, city string, limit int) ([]conn.POI, error) {
	c, ok := o.geo.Coords(city)
	if !ok {
		return nil, conn.ErrUnavailable
	}
	body := "data=" + url.QueryEscape(buildQuery(c))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := o.http.Do(req)
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
	return parsePOIs(raw, city, limit, o.provenance())
}

func (o *Overpass) provenance() conn.Provenance {
	now := o.now().UTC()
	return conn.Provenance{Source: sourceName, FetchedAt: now, ExpiresAt: now.Add(o.ttl), Estimated: false}
}

// overpassResp mirrors the Overpass JSON payload.
type overpassResp struct {
	Elements []struct {
		Lat  float64           `json:"lat"`
		Lon  float64           `json:"lon"`
		Tags map[string]string `json:"tags"`
	} `json:"elements"`
}

// parsePOIs turns an Overpass response into named, categorized POIs. It drops
// unnamed elements, de-duplicates by name, sorts by name for determinism, and
// caps at limit. Pure, so it is unit-tested without the network.
func parsePOIs(body []byte, city string, limit int, prov conn.Provenance) ([]conn.POI, error) {
	var r overpassResp
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("decode overpass response: %w", err)
	}
	seen := make(map[string]bool)
	out := make([]conn.POI, 0, len(r.Elements))
	for _, el := range r.Elements {
		name := strings.TrimSpace(el.Tags["name"])
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, conn.POI{
			City:       city,
			Name:       name,
			Category:   category(el.Tags),
			Lat:        el.Lat,
			Lng:        el.Lon,
			Image:      osmImage(el.Tags),
			Website:    firstNonEmpty(el.Tags["website"], el.Tags["url"], el.Tags["contact:website"]),
			Wiki:       wikiURL(el.Tags["wikipedia"]),
			Provenance: prov,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

// osmImage returns a free image URL from OSM tags: a direct `image` URL, or a
// Wikimedia Commons file resolved via Special:FilePath (returns the actual bytes).
func osmImage(tags map[string]string) string {
	if img := strings.TrimSpace(tags["image"]); strings.HasPrefix(img, "http") {
		return img
	}
	commons := strings.TrimSpace(tags["wikimedia_commons"])
	if strings.HasPrefix(commons, "File:") {
		file := strings.TrimPrefix(commons, "File:")
		return "https://commons.wikimedia.org/wiki/Special:FilePath/" + url.PathEscape(file) + "?width=480"
	}
	return ""
}

// wikiURL turns an OSM `wikipedia` tag ("en:Narikala Fortress") into a real
// article URL. Returns "" when the tag is missing or malformed.
func wikiURL(tag string) string {
	tag = strings.TrimSpace(tag)
	lang, title, ok := strings.Cut(tag, ":")
	if !ok || lang == "" || strings.TrimSpace(title) == "" {
		return ""
	}
	return "https://" + lang + ".wikipedia.org/wiki/" + url.PathEscape(strings.ReplaceAll(strings.TrimSpace(title), " ", "_"))
}

// category picks a human-friendly category from OSM tags.
func category(tags map[string]string) string {
	if t := tags["tourism"]; t != "" {
		return t
	}
	if h := tags["historic"]; h != "" {
		return "historic:" + h
	}
	return "poi"
}

var _ conn.POISource = (*Overpass)(nil)
