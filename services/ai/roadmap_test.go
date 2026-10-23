package ai

import (
	"context"
	"testing"
)

func TestStructuredRoadmap_SkeletonNightsAndCoordFallback(t *testing.T) {
	// Model returns city summaries + POIs keyed by day; one POI has no coords.
	llm := &MockLLM{Script: []Completion{{Content: "```json\n" + `{
		"cities":[
			{"city":"Tbilisi","summary":"Cozy old town.","best_season":"spring"},
			{"city":"Istanbul","summary":"Two continents.","best_season":"autumn"}
		],
		"days":[
			{"day":1,"city":"Tbilisi","title":"Old town","pois":[
				{"name":"Narikala Fortress","type":"sight","free":true,"lat":41.6878,"lng":44.8094},
				{"name":"Sulphur Baths","type":"spa","free":false,"price":"~€15"}
			]},
			{"day":2,"city":"Tbilisi","pois":[{"name":"Mtatsminda Park","free":false,"price":"~€5"}]},
			{"day":3,"city":"Istanbul","pois":[{"name":"Hagia Sophia","free":false,"price":"~€25"}]}
		]
	}` + "\n```"}}}

	coords := map[string]LatLng{
		"tbilisi":  {Lat: 41.7151, Lng: 44.8271},
		"istanbul": {Lat: 41.0082, Lng: 28.9784},
	}
	r, err := StructuredRoadmap(context.Background(), llm, []string{"Tbilisi", "Istanbul"}, []int{2, 1}, coords, "English")
	if err != nil {
		t.Fatalf("roadmap: %v", err)
	}
	// 2 nights Tbilisi + 1 night Istanbul => 3 days.
	if len(r.Days) != 3 {
		t.Fatalf("expected 3 days, got %d", len(r.Days))
	}
	if r.Days[0].City != "Tbilisi" || r.Days[2].City != "Istanbul" {
		t.Fatalf("day->city mapping wrong: %+v", []string{r.Days[0].City, r.Days[2].City})
	}
	// Nights + city coords are authoritative from the input.
	if len(r.Cities) != 2 || r.Cities[0].Nights != 2 || r.Cities[1].Nights != 1 {
		t.Fatalf("city nights wrong: %+v", r.Cities)
	}
	if r.Cities[0].Lat == 0 || r.Cities[0].Summary != "Cozy old town." {
		t.Fatalf("city info/coords not merged: %+v", r.Cities[0])
	}
	// POI with no coords falls back to the city centre.
	baths := r.Days[0].POIs[1]
	if baths.Name != "Sulphur Baths" || baths.Lat != 41.7151 {
		t.Fatalf("POI coord fallback failed: %+v", baths)
	}
	if baths.Price != "~€15" || baths.Free {
		t.Fatalf("paid POI fields lost: %+v", baths)
	}
}

func TestStructuredRoadmap_NoLLM(t *testing.T) {
	if _, err := StructuredRoadmap(context.Background(), nil, []string{"Baku"}, []int{1}, nil, "English"); err == nil {
		t.Fatal("expected an error without an LLM")
	}
}
