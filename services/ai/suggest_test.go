package ai

import (
	"context"
	"testing"
)

func TestSuggestDestinations_CleansOutput(t *testing.T) {
	// Model returns the origin, a duplicate and an empty entry — all dropped.
	llm := &MockLLM{Script: []Completion{
		{Content: "```json\n{\"cities\":[\"Baku\",\"Tbilisi\",\"Istanbul\",\"Tbilisi\",\"\"]}\n```"},
	}}
	names, err := SuggestDestinations(context.Background(), llm, "Baku", 600, 5)
	if err != nil {
		t.Fatalf("suggest: %v", err)
	}
	if len(names) != 2 || names[0] != "Tbilisi" || names[1] != "Istanbul" {
		t.Fatalf("expected [Tbilisi Istanbul] (origin/dupe/empty dropped), got %v", names)
	}
}

func TestSuggestDestinations_NoLLM(t *testing.T) {
	if _, err := SuggestDestinations(context.Background(), nil, "Baku", 600, 5); err == nil {
		t.Fatal("expected an error without an LLM")
	}
}
