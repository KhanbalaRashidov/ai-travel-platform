package dataflow

import (
	"context"
	"testing"

	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/cache"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/connectors/estimate"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/knowledge"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/planner"
)

func newTestPlanner() *PricePlanner {
	geo := knowledge.SeedGeocoder()
	est := estimate.New(geo, 0)
	fw := NewRouteFlywheel(cache.NewMemKV(), knowledge.NewMemStore(), est, 0)
	conv := planner.StaticConverter{Rates: map[string]float64{"EUR": 1, "USD": 0.92, "AZN": 0.53}}
	return NewPricePlanner(fw, est, geo, conv)
}

func TestPlan_SequencesAndPrices(t *testing.T) {
	p := newTestPlanner()
	res, err := p.Plan(context.Background(), "t1", PlanRequest{
		Cities: []string{"Baku", "Tbilisi", "Istanbul"}, Start: "Baku",
		Mode: "cheapest", Tier: "standard", Nights: 2, Currency: "EUR",
	})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if len(res.Order) != 3 || len(res.Legs) != 2 {
		t.Fatalf("expected 3 stops / 2 legs, got %v / %d", res.Order, len(res.Legs))
	}
	if res.Order[0] != "Baku" {
		t.Fatalf("start must lead the order, got %v", res.Order)
	}
	if res.Transport <= 0 || res.Hotels <= 0 || res.Activities <= 0 || res.Total <= 0 {
		t.Fatalf("all cost components should be positive: %+v", res)
	}
	if !res.Estimated {
		t.Fatal("estimator-backed plan must be flagged estimated")
	}
	for _, l := range res.Legs {
		if !l.Estimated || l.Source == "" {
			t.Fatalf("each leg must carry provenance: %+v", l)
		}
	}
}

func TestPlan_OrderGivenSkipsResequence(t *testing.T) {
	p := newTestPlanner()
	// A deliberately non-optimal order; with Order supplied it must be preserved.
	fixed := []string{"Istanbul", "Baku", "Tbilisi"}
	res, err := p.Plan(context.Background(), "t1", PlanRequest{
		Cities: []string{"Baku", "Tbilisi", "Istanbul"}, Start: "Baku",
		Mode: "cheapest", Tier: "budget", Nights: 1, Currency: "EUR",
		Order: fixed, Variant: "open_jaw",
	})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	for i := range fixed {
		if res.Order[i] != fixed[i] {
			t.Fatalf("supplied order must be preserved, got %v", res.Order)
		}
	}
	if res.Variant != "open_jaw" {
		t.Fatalf("variant should echo the request on re-price, got %q", res.Variant)
	}
}

func TestPlan_CurrencyConversion(t *testing.T) {
	p := newTestPlanner()
	req := PlanRequest{Cities: []string{"Baku", "Tbilisi"}, Start: "Baku", Mode: "flight", Tier: "standard", Nights: 1}
	eur, err := p.Plan(context.Background(), "t1", withCurrency(req, "EUR"))
	if err != nil {
		t.Fatal(err)
	}
	usd, err := p.Plan(context.Background(), "t1", withCurrency(req, "USD"))
	if err != nil {
		t.Fatal(err)
	}
	// EUR->USD uses rate 0.92 (1 EUR = 1/0.92 USD), so USD total must exceed EUR.
	if !(usd.Total > eur.Total) {
		t.Fatalf("USD total (%v) should exceed EUR total (%v)", usd.Total, eur.Total)
	}
	if usd.Currency != "USD" {
		t.Fatalf("currency not applied: %s", usd.Currency)
	}
}

func withCurrency(r PlanRequest, cur string) PlanRequest {
	r.Currency = cur
	return r
}
