package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// LatLng is a city's authoritative coordinate (from the plan, not the model).
type LatLng struct {
	Lat float64
	Lng float64
}

// POI is one location-based thing to see/do on a day. Coordinates are best-effort
// from the model and fall back to the city centre; the UI builds a real Google
// Maps link from the name. Paid items carry an APPROXIMATE "~"-marked price.
type POI struct {
	Name  string  `json:"name"`
	Type  string  `json:"type,omitempty"` // sight, museum, food, park, viewpoint, transfer…
	Desc  string  `json:"desc,omitempty"`
	Free  bool    `json:"free"`
	Price string  `json:"price,omitempty"`
	Lat   float64 `json:"lat,omitempty"`
	Lng   float64 `json:"lng,omitempty"`
}

// DayPlan is one day of the itinerary, anchored to a city and a sequence of POIs.
type DayPlan struct {
	Day   int    `json:"day"`
	City  string `json:"city"`
	Title string `json:"title,omitempty"`
	POIs  []POI  `json:"pois"`
}

// CityInfo is the per-city advice card (§1): nights, a short summary and the best
// season. Nights + coordinates are authoritative from the plan input.
type CityInfo struct {
	City       string  `json:"city"`
	Nights     int     `json:"nights"`
	Summary    string  `json:"summary,omitempty"`
	BestSeason string  `json:"best_season,omitempty"`
	Lat        float64 `json:"lat,omitempty"`
	Lng        float64 `json:"lng,omitempty"`
}

// Roadmap is the structured, day-by-day, location-based itinerary the UI renders
// as day cards + map pins (never raw markdown).
type Roadmap struct {
	Cities []CityInfo `json:"cities"`
	Days   []DayPlan  `json:"days"`
}

const roadmapSystem = "You are a budget, travel-loving planner building a LOCATION-BASED, day-by-day itinerary. " +
	"You are given an ordered list of days, each already assigned to a city, plus each city's nights. " +
	"For EACH given day number, fill 2–4 POIs (points of interest) to visit that day, in a sensible " +
	"walking/visiting order and grouped by proximity. For EACH city also give a warm 1–2 sentence summary " +
	"and a best_season. Each POI: name, type (sight/museum/food/park/viewpoint/transfer), a short desc, " +
	"free=true/false, an APPROXIMATE price like \"~€15\" for paid ones (always prefix ~), and best-effort " +
	"lat/lng coordinates (0 if unknown). Do NOT invent flight, hotel or transport prices. " +
	"Return STRICT JSON: {\"cities\":[{\"city\":\"\",\"summary\":\"\",\"best_season\":\"\"}]," +
	"\"days\":[{\"day\":<int>,\"city\":\"\",\"title\":\"\",\"pois\":[{\"name\":\"\",\"type\":\"\"," +
	"\"desc\":\"\",\"free\":true,\"price\":\"\",\"lat\":0,\"lng\":0}]}]}."

// daySkeleton builds the deterministic day->city assignment from the cities and
// their nights (at least one day per city). Days are numbered from 1.
func daySkeleton(cities []string, nights []int) []DayPlan {
	days := make([]DayPlan, 0)
	n := 1
	for i, c := range cities {
		d := 1
		if i < len(nights) && nights[i] > 1 {
			d = nights[i]
		}
		for k := 0; k < d; k++ {
			days = append(days, DayPlan{Day: n, City: c})
			n++
		}
	}
	return days
}

// StructuredRoadmap asks the LLM to fill a deterministic day skeleton with POIs
// and per-city advice, in the given language. Day/city structure and nights come
// from the input (authoritative); the LLM only supplies descriptive content.
// coords maps a lower-cased city name to its centre (for POI fallback + pins).
func StructuredRoadmap(ctx context.Context, llm LLM, cities []string, nights []int, coords map[string]LatLng, language string) (Roadmap, error) {
	if llm == nil {
		return Roadmap{}, fmt.Errorf("roadmap: no llm configured")
	}
	skeleton := daySkeleton(cities, nights)

	var sb strings.Builder
	sb.WriteString("Days (fill POIs for each):\n")
	for _, d := range skeleton {
		fmt.Fprintf(&sb, "  - day %d: %s\n", d.Day, d.City)
	}
	sb.WriteString("City nights:\n")
	for i, c := range cities {
		n := 1
		if i < len(nights) {
			n = nights[i]
		}
		fmt.Fprintf(&sb, "  - %s: %d\n", c, n)
	}
	if lang := strings.TrimSpace(language); lang != "" {
		fmt.Fprintf(&sb, "Write all text in %s (keep place and city names recognizable).", lang)
	}

	comp, err := completeStructured(ctx, llm, []Message{
		{Role: RoleSystem, Content: roadmapSystem},
		{Role: RoleUser, Content: sb.String()},
	})
	if err != nil {
		return Roadmap{}, err
	}
	var parsed Roadmap
	if err := json.Unmarshal([]byte(extractJSONObject(comp.Content)), &parsed); err != nil {
		return Roadmap{}, fmt.Errorf("roadmap: parse: %w (raw: %q)", err, snippet(comp.Content, 160))
	}

	return mergeRoadmap(skeleton, parsed, cities, nights, coords), nil
}

// mergeRoadmap stitches the model's content onto the authoritative skeleton:
// nights + city coords from the input, POIs matched to days by day number, and
// each POI's missing coordinate filled from its city centre.
func mergeRoadmap(skeleton []DayPlan, parsed Roadmap, cities []string, nights []int, coords map[string]LatLng) Roadmap {
	// City summaries/seasons by lower-cased name.
	type info struct{ summary, season string }
	cityInfo := map[string]info{}
	for _, c := range parsed.Cities {
		cityInfo[strings.ToLower(strings.TrimSpace(c.City))] = info{c.Summary, c.BestSeason}
	}
	// POIs by day number from the model.
	poisByDay := map[int][]POI{}
	titleByDay := map[int]string{}
	for _, d := range parsed.Days {
		poisByDay[d.Day] = d.POIs
		titleByDay[d.Day] = d.Title
	}

	out := Roadmap{}
	for i, c := range cities {
		key := strings.ToLower(strings.TrimSpace(c))
		ci := CityInfo{City: c, Summary: cityInfo[key].summary, BestSeason: cityInfo[key].season}
		if i < len(nights) {
			ci.Nights = nights[i]
		}
		if ll, ok := coords[key]; ok {
			ci.Lat, ci.Lng = ll.Lat, ll.Lng
		}
		out.Cities = append(out.Cities, ci)
	}
	for _, d := range skeleton {
		key := strings.ToLower(strings.TrimSpace(d.City))
		center, hasCenter := coords[key]
		pois := poisByDay[d.Day]
		for j := range pois {
			if (pois[j].Lat == 0 && pois[j].Lng == 0) && hasCenter {
				pois[j].Lat, pois[j].Lng = center.Lat, center.Lng
			}
		}
		out.Days = append(out.Days, DayPlan{Day: d.Day, City: d.City, Title: titleByDay[d.Day], POIs: pois})
	}
	return out
}
