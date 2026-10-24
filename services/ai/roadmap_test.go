package ai

import (
	"context"
	"testing"
)

// C3: the trip origin (0 nights) gets no city card and no day; Day 1 is the
// first real destination. Both the skeleton and a filled roadmap must honor it.
func TestRoadmap_OriginHasNoDaysOrCard(t *testing.T) {
	cities := []string{"Baku", "Tbilisi", "Istanbul"}
	nights := []int{0, 2, 1} // Baku = origin
	sk := SkeletonRoadmap(cities, nights, nil)

	if len(sk.Days) != 3 {
		t.Fatalf("expected 3 days (2 Tbilisi + 1 Istanbul), got %d", len(sk.Days))
	}
	if sk.Days[0].City != "Tbilisi" || sk.Days[0].Day != 1 {
		t.Fatalf("Day 1 must be the first destination, got %q day %d", sk.Days[0].City, sk.Days[0].Day)
	}
	for _, c := range sk.Cities {
		if c.City == "Baku" {
			t.Fatalf("origin must not get a city card: %+v", sk.Cities)
		}
	}
	if len(sk.Cities) != 2 {
		t.Fatalf("expected 2 destination cards, got %d", len(sk.Cities))
	}
}

func TestStructuredRoadmap_DistributesPOIsAndKeepsSkeleton(t *testing.T) {
	llm := &MockLLM{Script: []Completion{{Content: "```json\n" + `{
		"cities":[
			{"city":"Tbilisi","summary":"Cozy old town.","best_season":"spring"},
			{"city":"Istanbul","summary":"Two continents.","best_season":"autumn"}
		],
		"days":[
			{"day":1,"city":"Tbilisi","pois":[
				{"name":"Narikala","type":"sight","free":true},
				{"name":"Sulphur Baths","type":"spa","free":false,"price":"~€15"},
				{"name":"Mtatsminda","type":"park","free":false,"price":"~€5"},
				{"name":"Old Town Walk","type":"sight","free":true}
			]},
			{"day":3,"city":"Istanbul","pois":[{"name":"Hagia Sophia","free":false,"price":"~€25"}]}
		]
	}` + "\n```"}}}

	coords := map[string]LatLng{"tbilisi": {Lat: 41.7151, Lng: 44.8271}, "istanbul": {Lat: 41.0082, Lng: 28.9784}}
	r, err := StructuredRoadmap(context.Background(), llm, []string{"Baku", "Tbilisi", "Istanbul"}, []int{0, 2, 1}, coords, "English")
	if err != nil {
		t.Fatalf("roadmap: %v", err)
	}
	if len(r.Days) != 3 {
		t.Fatalf("expected 3 days, got %d", len(r.Days))
	}
	// Tbilisi's 4 POIs spread across its 2 days (2 + 2); no empty Tbilisi day.
	if len(r.Days[0].POIs) == 0 || len(r.Days[1].POIs) == 0 {
		t.Fatalf("Tbilisi days should both have POIs: %d, %d", len(r.Days[0].POIs), len(r.Days[1].POIs))
	}
	if len(r.Days[0].POIs)+len(r.Days[1].POIs) != 4 {
		t.Fatalf("expected 4 Tbilisi POIs distributed, got %d", len(r.Days[0].POIs)+len(r.Days[1].POIs))
	}
	// City coords + summary merged; POI coord fell back to city centre.
	if r.Cities[0].Lat == 0 || r.Cities[0].Summary != "Cozy old town." {
		t.Fatalf("city info not merged: %+v", r.Cities[0])
	}
	if r.Days[0].POIs[0].Lat != 41.7151 {
		t.Fatalf("POI coord fallback failed: %+v", r.Days[0].POIs[0])
	}
}

// D18: after backfill, no day may be left empty.
func TestRebalanceDays_FillsEveryDay(t *testing.T) {
	// 3 days in one city but the model only gave 2 POIs on day 1.
	r := Roadmap{
		Cities: []CityInfo{{City: "Tbilisi", Nights: 3}},
		Days: []DayPlan{
			{Day: 1, City: "Tbilisi", POIs: []POI{{Name: "Narikala"}, {Name: "Old Town"}}},
			{Day: 2, City: "Tbilisi"},
			{Day: 3, City: "Tbilisi"},
		},
	}
	extra := map[string][]POI{
		"tbilisi": {{Name: "Sulphur Baths"}, {Name: "Mtatsminda"}, {Name: "Narikala"} /*dup*/, {Name: "Rustaveli Ave"}},
	}
	coords := map[string]LatLng{"tbilisi": {Lat: 41.7151, Lng: 44.8271}}
	out := RebalanceDays(r, extra, coords, 4)
	if len(out.Days) != 3 {
		t.Fatalf("expected 3 days, got %d", len(out.Days))
	}
	for i, d := range out.Days {
		if len(d.POIs) == 0 {
			t.Fatalf("day %d is empty after rebalance: %+v", i+1, out.Days)
		}
		for _, p := range d.POIs {
			if p.Lat == 0 || p.Lng == 0 {
				t.Fatalf("POI coord not filled from centre: %+v", p)
			}
		}
	}
}

func TestStructuredRoadmap_FallsBackToSkeletonOnLLMError(t *testing.T) {
	llm := &MockLLM{Script: nil} // exhausted -> error
	r, err := StructuredRoadmap(context.Background(), llm, []string{"Baku", "Tbilisi"}, []int{0, 2}, nil, "English")
	if err == nil {
		t.Fatal("expected an error to be surfaced for logging")
	}
	// But a usable skeleton still comes back (never a hard failure).
	if len(r.Days) != 2 || r.Days[0].City != "Tbilisi" {
		t.Fatalf("skeleton fallback missing/wrong: %+v", r.Days)
	}
}
