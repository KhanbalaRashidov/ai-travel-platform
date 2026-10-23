package ai

import (
	"context"
	"testing"
)

type fakeTool struct {
	name string
	out  string
}

func (f fakeTool) Name() string { return f.name }
func (f fakeTool) Call(_ context.Context, _ map[string]any) (string, error) {
	return f.out, nil
}

func TestRun_GroundedFlow(t *testing.T) {
	llm := &MockLLM{Script: []Completion{
		{ToolCalls: []ToolCall{{ID: "1", Name: "price", Args: map[string]any{"from": "BAK", "to": "IST"}}}},
		{Content: "Estimated flight total is 120 EUR."},
	}}
	o := NewOrchestrator(llm, 8, 1, fakeTool{name: "price", out: "BAK->IST flight price: 120 EUR"})

	res, err := o.Run(context.Background(), "system", "plan BAK to IST")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Violations) != 0 {
		t.Fatalf("expected grounded result, got violations %+v", res.Violations)
	}
	if res.Steps != 2 || llm.Calls != 2 {
		t.Fatalf("expected 2 steps/calls, got steps=%d calls=%d", res.Steps, llm.Calls)
	}
}

func TestRun_HallucinationThenRegen(t *testing.T) {
	llm := &MockLLM{Script: []Completion{
		{ToolCalls: []ToolCall{{ID: "1", Name: "price"}}},
		{Content: "Total is 999 EUR."}, // fabricated -> must be rejected
		{Content: "Total is 120 EUR."}, // corrected
	}}
	o := NewOrchestrator(llm, 8, 1, fakeTool{name: "price", out: "flight price: 120 EUR"})

	res, err := o.Run(context.Background(), "system", "plan")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Violations) != 0 {
		t.Fatalf("expected regenerated grounded result, got %+v", res.Violations)
	}
	if res.Steps != 3 || llm.Calls != 3 {
		t.Fatalf("expected 3 steps/calls after regen, got steps=%d calls=%d", res.Steps, llm.Calls)
	}
}

func TestRun_HallucinationNoRegenBudget(t *testing.T) {
	llm := &MockLLM{Script: []Completion{
		{ToolCalls: []ToolCall{{ID: "1", Name: "price"}}},
		{Content: "Total is 999 EUR."},
	}}
	o := NewOrchestrator(llm, 8, 0, fakeTool{name: "price", out: "flight price: 120 EUR"})

	res, err := o.Run(context.Background(), "system", "plan")
	if err != nil {
		t.Fatal(err)
	}
	// With no regen budget the ungrounded price is surfaced, never silently trusted.
	if len(res.Violations) != 1 || res.Violations[0].Value != 999 {
		t.Fatalf("expected one surfaced violation of 999, got %+v", res.Violations)
	}
}
