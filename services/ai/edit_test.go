package ai

import (
	"context"
	"strings"
	"testing"
)

func state() EditState {
	return EditState{
		Cities: []string{"Baku", "Tbilisi", "Istanbul"},
		Start:  "Baku",
		Tier:   "vip",
		Mode:   "flight",
		Nights: 2,
	}
}

func hasChange(changed []string, want string) bool {
	for _, c := range changed {
		if c == want {
			return true
		}
	}
	return false
}

func TestApplyEdit_RemoveCityReassignsStart(t *testing.T) {
	s, changed := ApplyEdit(state(), TripEdit{RemoveCities: []string{"Baku"}})
	if contains(s.Cities, "Baku") {
		t.Fatal("Baku should be removed")
	}
	if s.Start != "Tbilisi" {
		t.Fatalf("start should move to the first remaining city, got %q", s.Start)
	}
	if !hasChange(changed, "removed:Baku") {
		t.Fatalf("missing change code: %v", changed)
	}
}

func TestApplyEdit_Cheaper(t *testing.T) {
	s, changed := ApplyEdit(state(), TripEdit{Cheaper: true})
	if s.Mode != "cheapest" {
		t.Fatalf("cheaper should set mode=cheapest, got %q", s.Mode)
	}
	if s.Tier != "standard" {
		t.Fatalf("cheaper should downgrade vip->standard, got %q", s.Tier)
	}
	if !hasChange(changed, "mode:cheapest") || !hasChange(changed, "tier:standard") {
		t.Fatalf("missing change codes: %v", changed)
	}
}

func TestApplyEdit_NightsDeltaAndClamp(t *testing.T) {
	s, _ := ApplyEdit(state(), TripEdit{NightsDelta: 1})
	if s.Nights != 3 {
		t.Fatalf("nights should be 3, got %d", s.Nights)
	}
	// Clamp at the upper bound.
	s2, _ := ApplyEdit(EditState{Nights: 10}, TripEdit{NightsDelta: 5})
	if s2.Nights != 10 {
		t.Fatalf("nights should clamp at 10, got %d", s2.Nights)
	}
}

func TestApplyEdit_AddCityAndSetters(t *testing.T) {
	s, changed := ApplyEdit(state(), TripEdit{AddCities: []string{"Yerevan"}, SetTier: "budget", SetMode: "train"})
	if !contains(s.Cities, "Yerevan") {
		t.Fatal("Yerevan should be added")
	}
	if s.Tier != "budget" || s.Mode != "train" {
		t.Fatalf("setters not applied: tier=%s mode=%s", s.Tier, s.Mode)
	}
	if !hasChange(changed, "added:Yerevan") {
		t.Fatalf("missing added code: %v", changed)
	}
}

func TestApplyEdit_InvalidSettersIgnored(t *testing.T) {
	s, changed := ApplyEdit(state(), TripEdit{SetTier: "luxury", SetMode: "teleport"})
	if s.Tier != "vip" || s.Mode != "flight" {
		t.Fatalf("invalid setters must be ignored, got tier=%s mode=%s", s.Tier, s.Mode)
	}
	if len(changed) != 0 {
		t.Fatalf("no changes expected, got %v", changed)
	}
}

func TestExtractEdit_ParsesJSON(t *testing.T) {
	llm := &MockLLM{Script: []Completion{
		{Content: "```json\n{\"cheaper\":true,\"remove_cities\":[\"Istanbul\"],\"nights_delta\":1}\n```"},
	}}
	e, err := ExtractEdit(context.Background(), llm, "make it cheaper, skip Istanbul, one more night", "cities: Baku, Tbilisi, Istanbul")
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if !e.Cheaper || e.NightsDelta != 1 || len(e.RemoveCities) != 1 || !strings.EqualFold(e.RemoveCities[0], "Istanbul") {
		t.Fatalf("unexpected edit: %+v", e)
	}
}
