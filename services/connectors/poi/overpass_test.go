package poi

import (
	"strings"
	"testing"
	"time"

	conn "github.com/KhanbalaRashidov/ai-travel-platform/services/connectors"
)

const fixture = `{
  "elements": [
    {"lat": 41.0, "lon": 28.9, "tags": {"name": "Hagia Sophia", "tourism": "attraction"}},
    {"lat": 41.1, "lon": 29.0, "tags": {"name": "Topkapi", "historic": "castle"}},
    {"lat": 41.2, "lon": 29.1, "tags": {"tourism": "museum"}},
    {"lat": 41.3, "lon": 29.2, "tags": {"name": "Hagia Sophia", "tourism": "attraction"}},
    {"lat": 41.4, "lon": 29.3, "tags": {"name": "Basilica Cistern", "tourism": "attraction"}}
  ]
}`

func TestParsePOIs_FiltersDedupSortsLimits(t *testing.T) {
	prov := conn.Provenance{Source: sourceName, FetchedAt: time.Now().UTC()}
	pois, err := parsePOIs([]byte(fixture), "Istanbul", 10, prov)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	// 5 elements: one unnamed dropped, one duplicate name dropped -> 3 unique.
	if len(pois) != 3 {
		t.Fatalf("expected 3 POIs, got %d (%+v)", len(pois), pois)
	}
	// Sorted by name for determinism.
	if pois[0].Name != "Basilica Cistern" || pois[1].Name != "Hagia Sophia" || pois[2].Name != "Topkapi" {
		t.Fatalf("not sorted by name: %v", []string{pois[0].Name, pois[1].Name, pois[2].Name})
	}
	// Category mapping from tags.
	if pois[2].Category != "historic:castle" {
		t.Fatalf("expected historic:castle category, got %q", pois[2].Category)
	}
	if pois[0].City != "Istanbul" || pois[0].Source != sourceName || pois[0].Estimated {
		t.Fatalf("bad provenance/city: %+v", pois[0].Provenance)
	}
}

func TestParsePOIs_Limit(t *testing.T) {
	pois, err := parsePOIs([]byte(fixture), "Istanbul", 2, conn.Provenance{})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(pois) != 2 {
		t.Fatalf("limit not applied, got %d", len(pois))
	}
}

func TestEnrichment_ImageWebsiteWiki(t *testing.T) {
	body := `{"elements":[
		{"lat":41.0,"lon":44.8,"tags":{"name":"Narikala","tourism":"attraction",
		  "wikimedia_commons":"File:Narikala fortress.jpg","website":"https://narikala.ge",
		  "wikipedia":"en:Narikala"}},
		{"lat":41.1,"lon":44.9,"tags":{"name":"Direct Image","tourism":"artwork",
		  "image":"https://example.org/a.jpg"}}
	]}`
	pois, err := parsePOIs([]byte(body), "Tbilisi", 10, conn.Provenance{})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var narikala, direct conn.POI
	for _, p := range pois {
		switch p.Name {
		case "Narikala":
			narikala = p
		case "Direct Image":
			direct = p
		}
	}
	if !strings.HasPrefix(narikala.Image, "https://commons.wikimedia.org/wiki/Special:FilePath/") {
		t.Fatalf("commons image not resolved: %q", narikala.Image)
	}
	if narikala.Website != "https://narikala.ge" {
		t.Fatalf("website lost: %q", narikala.Website)
	}
	if narikala.Wiki != "https://en.wikipedia.org/wiki/Narikala" {
		t.Fatalf("wiki URL wrong: %q", narikala.Wiki)
	}
	if direct.Image != "https://example.org/a.jpg" {
		t.Fatalf("direct image lost: %q", direct.Image)
	}
}

func TestBuildQuery_HasBBoxAndTags(t *testing.T) {
	q := buildQuery(conn.LatLng{Lat: 41.0, Lng: 28.9})
	if !strings.Contains(q, "out:json") || !strings.Contains(q, "tourism") || !strings.Contains(q, "historic") {
		t.Fatalf("query missing expected clauses: %s", q)
	}
}
