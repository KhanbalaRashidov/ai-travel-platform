package dataflow

import (
	"context"
	"testing"
	"time"

	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/cache"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/connectors/estimate"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/knowledge"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/planner"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/pricefetch"
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

func TestPlan_NightsFromDates(t *testing.T) {
	p := newTestPlanner()
	res, err := p.Plan(context.Background(), "t1", PlanRequest{
		Cities: []string{"Baku", "Tbilisi", "Istanbul"}, Start: "Baku",
		Mode: "cheapest", Tier: "standard", Currency: "EUR",
		StartDate: "2026-07-01", ReturnDate: "2026-07-07", // 6 nights
	})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if res.Nights != 6 {
		t.Fatalf("total nights should be 6 (dates), got %d", res.Nights)
	}
	if len(res.NightsByCity) != 3 {
		t.Fatalf("expected per-city nights for 3 cities, got %v", res.NightsByCity)
	}
	sum := 0
	for _, n := range res.NightsByCity {
		sum += n
	}
	if sum != 6 {
		t.Fatalf("per-city nights must sum to total (6), got %d (%v)", sum, res.NightsByCity)
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

func TestPlan_PerLegModeOverride(t *testing.T) {
	p := newTestPlanner()
	res, err := p.Plan(context.Background(), "t1", PlanRequest{
		Cities: []string{"Baku", "Tbilisi"}, Start: "Baku",
		Mode: "flight", Tier: "standard", Nights: 1, Currency: "EUR",
		Order:    []string{"Baku", "Tbilisi"},
		LegModes: map[string]string{"Baku|Tbilisi": "bus"},
	})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if len(res.Legs) != 1 || res.Legs[0].Mode != "bus" {
		t.Fatalf("per-leg override should force bus, got %+v", res.Legs)
	}
}

func TestPlan_OriginNoHotel(t *testing.T) {
	p := newTestPlanner()
	res, err := p.Plan(context.Background(), "t1", PlanRequest{
		Origin: "Baku", Cities: []string{"Tbilisi", "Istanbul"},
		Mode: "cheapest", Tier: "standard", Nights: 2, Currency: "EUR",
	})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if len(res.Order) == 0 || res.Order[0] != "Baku" {
		t.Fatalf("origin must start the route, got %v", res.Order)
	}
	if len(res.HotelLines) != 2 {
		t.Fatalf("origin takes no hotel -> 2 hotels for 2 destinations, got %d", len(res.HotelLines))
	}
	if res.NightsByCity[0] != 0 {
		t.Fatalf("origin must have 0 nights, got %d", res.NightsByCity[0])
	}
	for _, h := range res.HotelLines {
		if h.City == "Baku" {
			t.Fatal("origin must not appear in hotels")
		}
	}
}

func TestPlan_RoundTripReturnLeg(t *testing.T) {
	p := newTestPlanner()
	res, err := p.Plan(context.Background(), "t1", PlanRequest{
		Cities: []string{"Baku", "Tbilisi"}, Start: "Baku", Mode: "flight",
		Tier: "standard", Nights: 1, Currency: "EUR",
		Order: []string{"Baku", "Tbilisi"}, Variant: "round_trip",
	})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	// Round-trip = out + return => 2 legs for 2 cities (open-jaw would be 1).
	if len(res.Legs) != 2 {
		t.Fatalf("round-trip should include the return leg, got %d legs", len(res.Legs))
	}
	last := res.Legs[len(res.Legs)-1]
	if last.From != "Tbilisi" || last.To != "Baku" {
		t.Fatalf("return leg should be Tbilisi -> Baku, got %s -> %s", last.From, last.To)
	}
}

func TestPlan_PassengerScaling(t *testing.T) {
	p := newTestPlanner()
	mk := func(adults int) PlanResult {
		res, err := p.Plan(context.Background(), "t1", PlanRequest{
			Cities: []string{"Baku", "Tbilisi"}, Start: "Baku", Mode: "flight",
			Tier: "standard", Nights: 1, Currency: "EUR",
			Order: []string{"Baku", "Tbilisi"}, Adults: adults,
		})
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	one := mk(1)
	two := mk(2)
	if !(two.Transport > one.Transport*1.9) {
		t.Fatalf("2 adults should ~double transport: one=%v two=%v", one.Transport, two.Transport)
	}
	if two.Travelers != 2 || two.PerPerson <= 0 {
		t.Fatalf("per-person fields wrong: travelers=%d per=%v", two.Travelers, two.PerPerson)
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

// blockingFares is a FareResolver whose Quote never returns a result until the
// context is cancelled — simulating a slow cold fetch.
type blockingFares struct{ called chan struct{} }

func (b *blockingFares) Enabled() bool { return true }
func (b *blockingFares) Quote(ctx context.Context, _ pricefetch.LegQuery, _, _ string) (pricefetch.Candidate, []pricefetch.Candidate, bool) {
	select {
	case b.called <- struct{}{}:
	default:
	}
	<-ctx.Done() // block until the per-leg deadline fires
	return pricefetch.Candidate{}, nil, false
}

func TestPlan_FareDeadlineFallsBackToEstimate(t *testing.T) {
	p := newTestPlanner().WithFares(&blockingFares{called: make(chan struct{}, 4)}).
		WithFareDeadline(60 * time.Millisecond)

	start := time.Now()
	res, err := p.Plan(context.Background(), "t1", PlanRequest{
		Cities: []string{"Baku", "Tbilisi"}, Start: "Baku",
		Mode: "flight", Tier: "standard", Nights: 1, Currency: "EUR",
	})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	// The slow fare source is bounded, so the plan still returns using the
	// labeled estimate — and promptly (one leg ~ one deadline, not forever).
	if !res.Estimated || len(res.Legs) == 0 || !res.Legs[0].Estimated {
		t.Fatalf("expected estimate fallback, got %+v", res)
	}
	if res.Total <= 0 {
		t.Fatalf("plan should still be priced: %+v", res)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("deadline did not bound the fetch: took %s", elapsed)
	}
}
