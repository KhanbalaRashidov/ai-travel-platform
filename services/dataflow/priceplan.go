package dataflow

import (
	"context"
	"errors"
	"math"
	"strings"

	conn "github.com/KhanbalaRashidov/ai-travel-platform/services/connectors"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/planner"
)

// Errors returned by the priced planner.
var (
	ErrPlanTooFewCities = errors.New("need at least 2 known cities")
	ErrLegUnavailable   = errors.New("no transport available for a leg")
)

// This file makes the data flywheel the single source of truth for pricing: the
// whole plan (transport via the flywheel, hotels + activities via the estimator)
// is priced server-side with provenance, so manual planning, tier/mode re-price
// and chat refine all go through one path with consistent, labeled numbers.

// HotelResolver provides a nightly hotel quote (connectors/estimate satisfies it).
type HotelResolver interface {
	Hotel(ctx context.Context, city string, stars int) (conn.HotelQuote, error)
}

// activitiesPerNightEUR is the indicative per-night activities budget (estimate).
const activitiesPerNightEUR = 20.0

// tierStars maps a comfort tier to the representative hotel star rating.
var tierStars = map[string]int{"budget": 3, "standard": 4, "vip": 5}

var transportModes = []conn.TransportMode{conn.ModeFlight, conn.ModeTrain, conn.ModeBus}

// LegQuote is one priced leg with provenance for the UI.
type LegQuote struct {
	From       string  `json:"from"`
	To         string  `json:"to"`
	Mode       string  `json:"mode"`
	Price      float64 `json:"price"` // display currency
	Estimated  bool    `json:"estimated"`
	Source     string  `json:"source"`
	SourceTier string  `json:"source_tier"`
}

// PlanRequest is a full priced-plan request. When Order is supplied and matches
// the city set, the TSP step is skipped and that order is only re-priced
// (preserving the "tier/mode toggle never re-sequences" behavior).
type PlanRequest struct {
	Cities   []string `json:"cities"`
	Start    string   `json:"start"`
	Mode     string   `json:"mode"`
	Tier     string   `json:"tier"`
	Nights   int      `json:"nights"`
	Currency string   `json:"currency"`
	Order    []string `json:"order"`
	Variant  string   `json:"variant"`
}

// PlanResult is the fully priced plan, all amounts in the display currency.
type PlanResult struct {
	Order              []string   `json:"order"`
	Variant            string     `json:"variant"`
	Legs               []LegQuote `json:"legs"`
	Transport          float64    `json:"transport"`
	Hotels             float64    `json:"hotels"`
	Activities         float64    `json:"activities"`
	Total              float64    `json:"total"`
	Currency           string     `json:"currency"`
	HotelStars         int        `json:"hotel_stars"`
	Nights             int        `json:"nights"`
	ActivitiesPerNight float64    `json:"activities_per_night"` // display currency
	Estimated          bool       `json:"estimated"`
	AltVariant         string     `json:"alt_variant,omitempty"`
	AltCost            float64    `json:"alt_cost,omitempty"`
}

// PricePlanner composes the flywheel (transport), a hotel resolver (estimator)
// and the TSP engine into one priced-plan computation.
type PricePlanner struct {
	routes *RouteFlywheel
	hotels HotelResolver
	geo    conn.Geocoder
	conv   planner.Converter
}

// NewPricePlanner wires the priced planner.
func NewPricePlanner(routes *RouteFlywheel, hotels HotelResolver, geo conn.Geocoder, conv planner.Converter) *PricePlanner {
	return &PricePlanner{routes: routes, hotels: hotels, geo: geo, conv: conv}
}

// cheapestLeg resolves the chosen transport quote for a leg under the request's
// mode preference ("cheapest" picks the min across modes). Returns ok=false when
// no mode is viable.
func (p *PricePlanner) cheapestLeg(ctx context.Context, tenantID, from, to, mode string) (conn.TransportQuote, Tier, bool) {
	if mode != "" && mode != "cheapest" {
		q, tier, err := p.routes.Route(ctx, tenantID, from, to, conn.TransportMode(mode))
		if err != nil {
			return conn.TransportQuote{}, "", false
		}
		return q, tier, true
	}
	var best conn.TransportQuote
	var bestTier Tier
	found := false
	for _, m := range transportModes {
		q, tier, err := p.routes.Route(ctx, tenantID, from, to, m)
		if err != nil {
			continue
		}
		if !found || q.Price.Amount < best.Price.Amount {
			best, bestTier, found = q, tier, true
		}
	}
	return best, bestTier, found
}

// Plan prices a full trip. It resolves the city order (TSP, unless a matching
// Order is given), prices every leg through the flywheel, hotels + activities
// through the estimator, and converts everything to the display currency.
func (p *PricePlanner) Plan(ctx context.Context, tenantID string, req PlanRequest) (PlanResult, error) {
	cur := req.Currency
	if cur == "" {
		cur = "EUR"
	}
	nights := req.Nights
	if nights < 1 {
		nights = 1
	}

	// Keep only cities we can geolocate.
	known := make([]string, 0, len(req.Cities))
	for _, c := range req.Cities {
		if _, ok := p.geo.Coords(c); ok {
			known = append(known, c)
		}
	}
	if len(known) < 2 {
		return PlanResult{}, ErrPlanTooFewCities
	}

	startIdx := indexOfFold(known, req.Start)
	if startIdx < 0 {
		startIdx = 0
	}

	var order []string
	variant := req.Variant
	var altVariant string
	var altCostEUR float64

	if len(req.Order) > 0 && sameSet(req.Order, known) {
		// Re-price the given order; skip the TSP step.
		order = append([]string(nil), req.Order...)
		if variant == "" {
			variant = string(planner.OpenJaw)
		}
	} else {
		// Sequence via the engine on a flywheel-priced cost matrix.
		m := make(planner.Matrix, len(known))
		for i := range known {
			m[i] = make([]float64, len(known))
			for j := range known {
				if i == j {
					continue
				}
				q, _, ok := p.cheapestLeg(ctx, tenantID, known[i], known[j], req.Mode)
				if !ok {
					m[i][j] = math.MaxFloat64
					continue
				}
				m[i][j] = q.Price.Amount
			}
		}
		best, alt, err := planner.Optimize(m, startIdx, -1)
		if err != nil {
			return PlanResult{}, err
		}
		order = make([]string, len(best.Order))
		for i, idx := range best.Order {
			order[i] = known[idx]
		}
		variant = string(best.Variant)
		altVariant, altCostEUR = string(alt.Variant), alt.Cost
	}

	// Price each leg through the flywheel.
	legs := make([]LegQuote, 0, len(order))
	priced := make([]planner.PricedLeg, 0, len(order))
	anyEstimated := false
	for i := 0; i+1 < len(order); i++ {
		q, tier, ok := p.cheapestLeg(ctx, tenantID, order[i], order[i+1], req.Mode)
		if !ok {
			return PlanResult{}, ErrLegUnavailable
		}
		disp, err := p.conv.Convert(q.Price.Amount, q.Price.Currency, cur)
		if err != nil {
			return PlanResult{}, err
		}
		if q.Estimated {
			anyEstimated = true
		}
		legs = append(legs, LegQuote{
			From: order[i], To: order[i+1], Mode: string(q.Mode),
			Price: round2(disp), Estimated: q.Estimated, Source: q.Source, SourceTier: string(tier),
		})
		priced = append(priced, planner.PricedLeg{From: i, To: i + 1, Mode: q.Mode, Price: q.Price})
	}

	// Hotels + activities via the estimator.
	stars := tierStars[req.Tier]
	if stars == 0 {
		stars = 4
	}
	hotels := make([]planner.PricedHotel, 0, len(order))
	for i, city := range order {
		h, err := p.hotels.Hotel(ctx, city, stars)
		if err != nil {
			return PlanResult{}, err
		}
		if h.Estimated {
			anyEstimated = true
		}
		hotels = append(hotels, planner.PricedHotel{City: i, Stars: h.Stars, Nights: nights, Nightly: h.Nightly})
	}
	activitiesEUR := activitiesPerNightEUR * float64(nights) * float64(len(order))
	anyEstimated = true // activities are always an estimate

	bd, err := planner.Price(planner.PlanInput{
		Order:           rangeN(len(order)),
		Legs:            priced,
		Hotels:          hotels,
		ActivitiesTotal: conn.Money{Amount: activitiesEUR, Currency: "EUR"},
		DisplayCurrency: cur,
	}, p.conv)
	if err != nil {
		return PlanResult{}, err
	}

	actPerNightDisp, _ := p.conv.Convert(activitiesPerNightEUR, "EUR", cur)
	res := PlanResult{
		Order: order, Variant: variant, Legs: legs,
		Transport: bd.Transport, Hotels: bd.Hotels, Activities: bd.Activities, Total: bd.Total,
		Currency: cur, HotelStars: stars, Nights: nights, ActivitiesPerNight: round2(actPerNightDisp),
		Estimated: anyEstimated,
	}
	if altVariant != "" {
		if c, err := p.conv.Convert(altCostEUR, "EUR", cur); err == nil {
			res.AltVariant, res.AltCost = altVariant, round2(c)
		}
	}
	return res, nil
}

func indexOfFold(list []string, v string) int {
	for i, x := range list {
		if strings.EqualFold(strings.TrimSpace(x), strings.TrimSpace(v)) {
			return i
		}
	}
	return -1
}

func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	return planner.CitySetKey(lowerAll(a)) == planner.CitySetKey(lowerAll(b))
}

func lowerAll(xs []string) []string {
	out := make([]string, len(xs))
	for i, x := range xs {
		out[i] = strings.ToLower(strings.TrimSpace(x))
	}
	return out
}

func rangeN(n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = i
	}
	return out
}

func round2(f float64) float64 { return math.Round(f*100) / 100 }
