package ai

import (
	"context"
	"testing"
)

func TestPopularPOIs_ParsesAndCaps(t *testing.T) {
	llm := &MockLLM{Script: []Completion{{Content: "```json\n" + `{"pois":[
		{"name":"Narikala Fortress","category":"sight","desc":"Old citadel.","free":true},
		{"name":"Sulphur Baths","category":"spa","desc":"Historic baths.","free":false,"price":"~€15"},
		{"name":"Mtatsminda Park","category":"park","free":false,"price":"~€5"}
	]}` + "\n```"}}}

	got, err := PopularPOIs(context.Background(), llm, "Tbilisi", "English", 2)
	if err != nil {
		t.Fatalf("poi: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("limit not applied, got %d", len(got))
	}
	if got[0].Name != "Narikala Fortress" || !got[0].Free {
		t.Fatalf("first POI parsed wrong: %+v", got[0])
	}
	if got[1].Price != "~€15" {
		t.Fatalf("paid price lost: %+v", got[1])
	}
}

func TestPopularPOIs_NoLLM(t *testing.T) {
	if _, err := PopularPOIs(context.Background(), nil, "Baku", "English", 5); err == nil {
		t.Fatal("expected an error without an LLM")
	}
}
