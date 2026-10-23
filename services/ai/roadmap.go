package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// roadmapMaxTokens is the output cap for the roadmap call — large, because a
// day-by-day JSON with POIs is far bigger than the 1k default (a too-small cap
// truncates the JSON and the parse fails).
const roadmapMaxTokens = 4096

// LatLng is a city's authoritative coordinate (from the plan, not the model).
type LatLng struct {
	Lat float64
	Lng float64
}

// POI is one location-based thing to see/do on a day. Coordinates fall back to
// the city centre; the UI builds a real Google Maps link from the name. Paid
// items carry an APPROXIMATE "~"-marked price.
type POI struct {
	Name  string  `json:"name"`
	Type  string  `json:"type,omitempty"`
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

// CityInfo is the per-city advice card (§1): nights, a short summary and the
// best season. Nights + coordinates are authoritative from the plan input.
type CityInfo struct {
	City       string  `json:"city"`
	Nights     int     `json:"nights"`
	Summary    string  `json:"summary,omitempty"`
	BestSeason string  `json:"best_season,omitempty"`
	Lat        float64 `json:"lat,omitempty"`
	Lng        float64 `json:"lng,omitempty"`
}

// Roadmap is the structured, day-by-day, location-based itinerary the UI renders
// as a connected timeline + map (never raw markdown).
type Roadmap struct {
	Cities []CityInfo `json:"cities"`
	Days   []DayPlan  `json:"days"`
}

const roadmapSystem = "You are a budget, travel-loving planner building a LOCATION-BASED, day-by-day itinerary. " +
	"You are given an ordered list of days, each already assigned to a city, plus each city's nights. " +
	"For EACH given day number, fill 2–4 POIs (points of interest) to visit that day, in a sensible " +
	"walking/visiting order grouped by proximity. For EACH city also give a warm 1–2 sentence summary and " +
	"a best_season. Each POI: name, type (sight/museum/food/park/viewpoint/transfer), a short desc, " +
	"free=true/false, and an APPROXIMATE price like \"~€15\" for paid ones (always prefix ~; free ones omit it). " +
	"Do NOT invent flight, hotel or transport prices. " +
	"Return STRICT JSON: {\"cities\":[{\"city\":\"\",\"summary\":\"\",\"best_season\":\"\"}]," +
	"\"days\":[{\"day\":<int>,\"city\":\"\",\"title\":\"\",\"pois\":[{\"name\":\"\",\"type\":\"\"," +
	"\"desc\":\"\",\"free\":true,\"price\":\"\"}]}]}."

// effectiveNights returns the day count for city i: its nights, or 1 when the
// nights slice is short. A city with explicit 0 nights (e.g. the trip origin)
// gets 0 days — it is a departure point, not a stay.
func effectiveNights(i int, nights []int) int {
	if i >= len(nights) {
		return 1
	}
	if nights[i] <= 0 {
		return 0
	}
	return nights[i]
}

// daySkeleton builds the deterministic day->city assignment. Origin-style cities
// (0 nights) are skipped, so Day 1 is always the first real destination.
func daySkeleton(cities []string, nights []int) []DayPlan {
	days := make([]DayPlan, 0)
	n := 1
	for i, c := range cities {
		for k := 0; k < effectiveNights(i, nights); k++ {
			days = append(days, DayPlan{Day: n, City: c})
			n++
		}
	}
	return days
}

// SkeletonRoadmap builds the deterministic roadmap with no model content: city
// cards (nights + coords, destinations only) and empty day shells. It is the
// never-fail fallback when the LLM is unavailable, times out or returns garbage;
// the UI then fills POIs lazily per city.
func SkeletonRoadmap(cities []string, nights []int, coords map[string]LatLng) Roadmap {
	out := Roadmap{}
	for i, c := range cities {
		if effectiveNights(i, nights) == 0 {
			continue // origin: no card, no days
		}
		ci := CityInfo{City: c, Nights: effectiveNights(i, nights)}
		if ll, ok := coords[strings.ToLower(strings.TrimSpace(c))]; ok {
			ci.Lat, ci.Lng = ll.Lat, ll.Lng
		}
		out.Cities = append(out.Cities, ci)
	}
	out.Days = daySkeleton(cities, nights)
	return out
}

// StructuredRoadmap asks the LLM to fill a deterministic day skeleton with POIs
// and per-city advice, in the given language. Day/city structure and nights come
// from the input (authoritative); the LLM only supplies descriptive content.
// coords maps a lower-cased city name to its centre (for POI fallback + pins).
// On ANY LLM/parse failure it returns the deterministic skeleton plus the error,
// so the caller can serve a usable roadmap (never a hard failure).
func StructuredRoadmap(ctx context.Context, llm LLM, cities []string, nights []int, coords map[string]LatLng, language string) (Roadmap, error) {
	skeleton := SkeletonRoadmap(cities, nights, coords)
	if llm == nil {
		return skeleton, fmt.Errorf("roadmap: no llm configured")
	}

	var sb strings.Builder
	sb.WriteString("Days (fill POIs for each):\n")
	for _, d := range skeleton.Days {
		fmt.Fprintf(&sb, "  - day %d: %s\n", d.Day, d.City)
	}
	sb.WriteString("City nights:\n")
	for _, c := range skeleton.Cities {
		fmt.Fprintf(&sb, "  - %s: %d\n", c.City, c.Nights)
	}
	if lang := strings.TrimSpace(language); lang != "" {
		fmt.Fprintf(&sb, "Write all text in %s (keep place and city names recognizable).", lang)
	}

	comp, err := completeStructured(WithMaxTokens(ctx, roadmapMaxTokens), llm, []Message{
		{Role: RoleSystem, Content: roadmapSystem},
		{Role: RoleUser, Content: sb.String()},
	})
	if err != nil {
		return skeleton, err
	}
	var parsed Roadmap
	if err := json.Unmarshal([]byte(extractJSONObject(comp.Content)), &parsed); err != nil {
		return skeleton, fmt.Errorf("roadmap: parse: %w (raw: %q)", err, snippet(comp.Content, 160))
	}

	return mergeRoadmap(skeleton, parsed, coords), nil
}

// mergeRoadmap stitches the model's content onto the authoritative skeleton.
// City summaries/seasons are matched by name. POIs are grouped by city and
// distributed across that city's skeleton days (robust to the model mis-
// numbering days), and each POI's missing coordinate is filled from its city
// centre. The skeleton's day/city/nights are never overridden.
func mergeRoadmap(skeleton Roadmap, parsed Roadmap, coords map[string]LatLng) Roadmap {
	// City summary/season by lower-cased name.
	type info struct{ summary, season string }
	cityInfo := map[string]info{}
	for _, c := range parsed.Cities {
		cityInfo[strings.ToLower(strings.TrimSpace(c.City))] = info{c.Summary, c.BestSeason}
	}
	// All model POIs grouped by city (lower-cased).
	poolByCity := map[string][]POI{}
	for _, d := range parsed.Days {
		key := strings.ToLower(strings.TrimSpace(d.City))
		poolByCity[key] = append(poolByCity[key], d.POIs...)
	}

	out := Roadmap{}
	for _, c := range skeleton.Cities {
		key := strings.ToLower(strings.TrimSpace(c.City))
		c.Summary, c.BestSeason = cityInfo[key].summary, cityInfo[key].season
		out.Cities = append(out.Cities, c)
	}

	// How many skeleton days each city has (to spread its POIs across them).
	daysPerCity := map[string]int{}
	for _, d := range skeleton.Days {
		daysPerCity[strings.ToLower(strings.TrimSpace(d.City))]++
	}
	cursor := map[string]int{} // per-city day index while distributing

	for _, d := range skeleton.Days {
		key := strings.ToLower(strings.TrimSpace(d.City))
		center, hasCenter := coords[key]
		pois := sliceForDay(poolByCity[key], daysPerCity[key], cursor[key])
		cursor[key]++
		for j := range pois {
			if pois[j].Lat == 0 && pois[j].Lng == 0 && hasCenter {
				pois[j].Lat, pois[j].Lng = center.Lat, center.Lng
			}
		}
		out.Days = append(out.Days, DayPlan{Day: d.Day, City: d.City, POIs: pois})
	}
	return out
}

// sliceForDay returns the share of a city's POI pool for its dayIndex-th day,
// splitting the pool as evenly as possible across totalDays days.
func sliceForDay(pool []POI, totalDays, dayIndex int) []POI {
	if totalDays <= 0 || len(pool) == 0 {
		return nil
	}
	per := len(pool) / totalDays
	rem := len(pool) % totalDays
	// Days before dayIndex consumed `per` (+1 for each of the first `rem`).
	start := 0
	for i := 0; i < dayIndex; i++ {
		start += per
		if i < rem {
			start++
		}
	}
	end := start + per
	if dayIndex < rem {
		end++
	}
	if start >= len(pool) {
		return nil
	}
	if end > len(pool) {
		end = len(pool)
	}
	return pool[start:end]
}
