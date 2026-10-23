package ai

import (
	"context"
	"testing"
)

func TestExtractConstraints_Basic(t *testing.T) {
	llm := &MockLLM{Script: []Completion{{
		Content: `{"cities":["Baku","Istanbul"],"days":7,"budget_eur":1200,"tier":"VIP","transport_pref":"train","group_size":2,"interests":["food"]}`,
	}}}
	c, err := ExtractConstraints(context.Background(), llm, "7 days Baku and Istanbul, VIP, by train, 2 people")
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Cities) != 2 || c.Days != 7 || c.BudgetEUR != 1200 {
		t.Fatalf("bad extract: %+v", c)
	}
	if c.Tier != "vip" || c.TransportPref != "train" || c.GroupSize != 2 {
		t.Fatalf("normalization wrong: %+v", c)
	}
	if c.SuggestCities || c.Tempo != "relaxed" {
		t.Fatalf("defaults wrong: %+v", c)
	}
}

func TestExtractConstraints_DefaultsAndFences(t *testing.T) {
	llm := &MockLLM{Script: []Completion{{Content: "```json\n{\"days\":3}\n```"}}}
	c, err := ExtractConstraints(context.Background(), llm, "3 days somewhere cheap")
	if err != nil {
		t.Fatal(err)
	}
	if c.Days != 3 || c.Tier != "standard" || c.TransportPref != "cheapest" || c.GroupSize != 1 {
		t.Fatalf("defaults not applied: %+v", c)
	}
	if !c.SuggestCities {
		t.Fatalf("no cities -> suggest_cities must be true: %+v", c)
	}
}

func TestExtractConstraints_InvalidJSON(t *testing.T) {
	llm := &MockLLM{Script: []Completion{{Content: "sorry, I can't"}}}
	if _, err := ExtractConstraints(context.Background(), llm, "x"); err == nil {
		t.Fatal("expected parse error for non-JSON reply")
	}
}
