package main

import (
	"context"
	"strings"
	"testing"

	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/cache"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/connectors/estimate"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/dataflow"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/knowledge"
)

func TestQuoteTool_GroundedEstimate(t *testing.T) {
	geo := knowledge.SeedGeocoder()
	fw := dataflow.NewRouteFlywheel(cache.NewMemKV(), knowledge.NewMemStore(), estimate.New(geo, 0), 0)
	tool := NewQuoteTool(fw, "t1")

	out, err := tool.Call(context.Background(), map[string]any{"from": "Baku", "to": "Tbilisi", "mode": "flight"})
	if err != nil {
		t.Fatalf("quote: %v", err)
	}
	if !strings.Contains(out, "EUR") || !strings.Contains(out, "estimated") {
		t.Fatalf("quote should be a grounded, labeled estimate: %q", out)
	}
}

func TestQuoteTool_RequiresEndpoints(t *testing.T) {
	geo := knowledge.SeedGeocoder()
	fw := dataflow.NewRouteFlywheel(cache.NewMemKV(), knowledge.NewMemStore(), estimate.New(geo, 0), 0)
	tool := NewQuoteTool(fw, "t1")
	if _, err := tool.Call(context.Background(), map[string]any{"from": "Baku"}); err == nil {
		t.Fatal("expected an error when 'to' is missing")
	}
}

func TestSequenceTool_CheapestOrder(t *testing.T) {
	tool := NewSequenceTool(estimate.New(knowledge.SeedGeocoder(), 0))
	out, err := tool.Call(context.Background(), map[string]any{
		"cities": []any{"Baku", "Tbilisi", "Istanbul"},
		"start":  "Baku",
	})
	if err != nil {
		t.Fatalf("sequence: %v", err)
	}
	if !strings.Contains(out, "Cheapest order") || !strings.Contains(out, "->") || !strings.Contains(out, "EUR") {
		t.Fatalf("unexpected sequence output: %q", out)
	}
}

func TestSequenceTool_NeedsTwoCities(t *testing.T) {
	tool := NewSequenceTool(estimate.New(knowledge.SeedGeocoder(), 0))
	if _, err := tool.Call(context.Background(), map[string]any{"cities": []any{"Baku"}}); err == nil {
		t.Fatal("expected an error with fewer than 2 cities")
	}
}
