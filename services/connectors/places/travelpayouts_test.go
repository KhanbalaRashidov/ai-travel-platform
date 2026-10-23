package places

import (
	"testing"
	"time"
)

const fixture = `[
  {"code":"BAK","name":"Baku","country_name":"Azerbaijan","type":"city","coordinates":{"lat":40.37,"lon":49.89}},
  {"code":"GYD","name":"Heydar Aliyev","country_name":"Azerbaijan","type":"airport","coordinates":{"lat":40.46,"lon":50.05}},
  {"code":"XXX","name":"","country_name":"Nowhere","type":"city","coordinates":{"lat":1,"lon":1}},
  {"code":"ZZZ","name":"No Coords","country_name":"Nowhere","type":"city","coordinates":{"lat":0,"lon":0}}
]`

func TestParsePlaces_FiltersAndMaps(t *testing.T) {
	ps := parsePlaces([]byte(fixture), time.Now().UTC(), 10)
	// Unnamed + zero-coord entries dropped -> 2 remain.
	if len(ps) != 2 {
		t.Fatalf("expected 2 places, got %d (%+v)", len(ps), ps)
	}
	if ps[0].Code != "BAK" || ps[0].Name != "Baku" || ps[0].Kind != "city" {
		t.Fatalf("city not mapped: %+v", ps[0])
	}
	if ps[0].Lat == 0 || ps[0].Lng == 0 {
		t.Fatalf("coordinates not mapped: %+v", ps[0])
	}
	if ps[1].Kind != "airport" {
		t.Fatalf("airport kind not mapped: %+v", ps[1])
	}
	if ps[0].Source != sourceName {
		t.Fatalf("provenance source: %q", ps[0].Source)
	}
}

func TestParsePlaces_Limit(t *testing.T) {
	ps := parsePlaces([]byte(fixture), time.Now().UTC(), 1)
	if len(ps) != 1 {
		t.Fatalf("limit not applied, got %d", len(ps))
	}
}

func TestParsePlaces_Garbage(t *testing.T) {
	if ps := parsePlaces([]byte("not json"), time.Now().UTC(), 10); ps != nil {
		t.Fatalf("garbage should yield nil, got %+v", ps)
	}
}
