package dataflow

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	conn "github.com/KhanbalaRashidov/ai-travel-platform/services/connectors"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/planner"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/pricefetch"
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

// HotelLine is one priced hotel for the detailed cost panel.
type HotelLine struct {
	City      string  `json:"city"`
	Stars     int     `json:"stars"`
	Name      string  `json:"name,omitempty"` // real hotel name when a source has one
	Nights    int     `json:"nights"`
	Nightly   float64 `json:"nightly"` // display currency
	Total     float64 `json:"total"`   // display currency
	Estimated bool    `json:"estimated"`
	Source    string  `json:"source"`
	URL       string  `json:"url"`
	FetchedAt string  `json:"fetched_at,omitempty"`
	// Alts are the other star tiers for this city (cheapest first), so the user
	// can switch a single city's hotel and re-price live (D6). Nightly/Total are
	// display-currency estimates; the chosen one is marked.
	Alts []HotelAlt `json:"alts,omitempty"`
}

// HotelAlt is one selectable hotel option (a star tier) for a city.
type HotelAlt struct {
	Stars   int     `json:"stars"`
	Nightly float64 `json:"nightly"` // display currency
	Total   float64 `json:"total"`   // display currency
	Chosen  bool    `json:"chosen"`
}

// LegAlt is one source's price for a leg, with its real deep-link (the
// provenance chain: the chosen source + the others that were checked).
type LegAlt struct {
	Source string  `json:"source"`
	Price  float64 `json:"price"` // display currency
	URL    string  `json:"url"`
	Chosen bool    `json:"chosen"`

	// Optional fare details, shown only when the source stated them.
	Operator string `json:"operator,omitempty"`
	Depart   string `json:"depart,omitempty"`
	Duration string `json:"duration,omitempty"`
	Stops    string `json:"stops,omitempty"`
	Baggage  string `json:"baggage,omitempty"`
}

// LegQuote is one priced leg with provenance for the UI.
type LegQuote struct {
	From         string   `json:"from"`
	To           string   `json:"to"`
	Mode         string   `json:"mode"`
	Price        float64  `json:"price"` // display currency
	Estimated    bool     `json:"estimated"`
	Source       string   `json:"source"`
	SourceTier   string   `json:"source_tier"`
	URL          string   `json:"url"` // real, route-prefilled deep-link
	Alternatives []LegAlt `json:"alternatives,omitempty"`
	FetchedAt    string   `json:"fetched_at,omitempty"`
	// Chosen-fare detail, surfaced inline on the route (D3). Populated ONLY from
	// the source that actually stated them — never invented; empty => "~estimate".
	Operator string `json:"operator,omitempty"`
	Depart   string `json:"depart,omitempty"`
	Duration string `json:"duration,omitempty"`
	Stops    string `json:"stops,omitempty"`
	Baggage  string `json:"baggage,omitempty"`
	// RequestedMode is the mode the user asked for when it was not available for
	// this leg (e.g. a bus across a sea); Mode then holds the substitute used.
	RequestedMode   string `json:"requested_mode,omitempty"`
	ModeUnavailable bool   `json:"mode_unavailable,omitempty"`
}

// PlaceCoord is a city the client selected via autocomplete, with coordinates,
// so the planner can price any destination (not just a seeded list).
type PlaceCoord struct {
	Name string  `json:"name"`
	Lat  float64 `json:"lat"`
	Lng  float64 `json:"lng"`
}

// PlanRequest is a full priced-plan request. When Order is supplied and matches
// the city set, the TSP step is skipped and that order is only re-priced
// (preserving the "tier/mode toggle never re-sequences" behavior). Places (with
// coordinates) + dates + passengers come from the search form; dates/passengers
// are pass-through for the real-fare connector.
type PlanRequest struct {
	Cities    []string     `json:"cities"`
	Places    []PlaceCoord `json:"places"`
	Origin    string       `json:"origin"`     // departure city; start, no hotel/nights
	BudgetEUR float64      `json:"budget_eur"` // for minimal mode / budget fit
	Start     string       `json:"start"`
	Mode      string       `json:"mode"`
	Tier      string       `json:"tier"`
	Nights    int          `json:"nights"`
	Currency  string       `json:"currency"`
	Order     []string     `json:"order"`
	Variant   string       `json:"variant"`
	// LegModes overrides the transport mode for specific legs, keyed by
	// "from|to" (used by the per-leg alternatives selector). Pricing-only — it
	// never changes the sequence.
	LegModes map[string]string `json:"leg_modes"`

	StartDate  string `json:"start_date"`
	ReturnDate string `json:"return_date"`
	Adults     int    `json:"adults"`
	Children   int    `json:"children"`
	Infants    int    `json:"infants"`

	// CityNights overrides the per-destination night count, keyed by city name
	// (the per-city stepper). Cities without an entry use the date-driven
	// allocation / AI-recommended default.
	CityNights map[string]int `json:"city_nights"`

	// CityStars overrides the hotel star tier for a specific city (the per-city
	// hotel selector, D6), keyed by city name. Cities without an entry use the
	// global tier. Values are clamped to 2..5.
	CityStars map[string]int `json:"city_stars"`
}

// Geocoder builds a case-insensitive geocoder from the request's places, so the
// planner can resolve any selected city's coordinates.
func (r PlanRequest) Geocoder() conn.Geocoder {
	m := make(map[string]conn.LatLng, len(r.Places))
	for _, p := range r.Places {
		if p.Name != "" {
			m[p.Name] = conn.LatLng{Lat: p.Lat, Lng: p.Lng}
		}
	}
	return conn.NewStaticGeocoder(m)
}

// PlanResult is the fully priced plan, all amounts in the display currency.
type PlanResult struct {
	Order              []string    `json:"order"`
	Variant            string      `json:"variant"`
	Legs               []LegQuote  `json:"legs"`
	HotelLines         []HotelLine `json:"hotel_lines"`
	Transport          float64     `json:"transport"`
	Hotels             float64     `json:"hotels"`
	Activities         float64     `json:"activities"`
	Total              float64     `json:"total"`
	Currency           string      `json:"currency"`
	HotelStars         int         `json:"hotel_stars"`
	Nights             int         `json:"nights"`         // total nights
	NightsByCity       []int       `json:"nights_by_city"` // aligned to Order
	Travelers          int         `json:"travelers"`
	PerPerson          float64     `json:"per_person"`           // total / travelers
	ActivitiesPerNight float64     `json:"activities_per_night"` // display currency
	Estimated          bool        `json:"estimated"`
	AltVariant         string      `json:"alt_variant,omitempty"`
	AltCost            float64     `json:"alt_cost,omitempty"`
	AltOrder           []string    `json:"alt_order,omitempty"`          // alternative's city sequence
	AltNightsByCity    []int       `json:"alt_nights_by_city,omitempty"` // aligned to AltOrder
}

// FareResolver is the real multi-source price lookup (pricefetch.Resolver
// satisfies it). When it returns a quote, it is preferred over the estimator.
type FareResolver interface {
	Enabled() bool
	Quote(ctx context.Context, q pricefetch.LegQuery, display, tenantID string) (pricefetch.Candidate, []pricefetch.Candidate, bool)
}

// PricePlanner composes the real fare resolver (preferred), the flywheel
// (estimator fallback), a hotel resolver and the TSP engine into one priced plan.
type PricePlanner struct {
	routes *RouteFlywheel
	hotels HotelResolver
	geo    conn.Geocoder
	conv   planner.Converter

	fares        FareResolver  // optional real-price source
	fareDeadline time.Duration // per-leg cold-fetch budget (0 = unbounded)
	tripDate     string        // per-call context for the fare resolver
	tripAdults   int
}

// NewPricePlanner wires the priced planner.
func NewPricePlanner(routes *RouteFlywheel, hotels HotelResolver, geo conn.Geocoder, conv planner.Converter) *PricePlanner {
	return &PricePlanner{routes: routes, hotels: hotels, geo: geo, conv: conv}
}

// WithFares attaches a real multi-source fare resolver (preferred over estimate).
func (p *PricePlanner) WithFares(f FareResolver) *PricePlanner { p.fares = f; return p }

// WithFareDeadline caps how long a single leg may spend on the real (cold) fetch
// before falling back to the labeled estimate, so a plan always returns promptly
// (§6 progressive). Cache hits are unaffected; <=0 means unbounded.
func (p *PricePlanner) WithFareDeadline(d time.Duration) *PricePlanner {
	p.fareDeadline = d
	return p
}

// legQuote builds a LegQuote with a REAL deep-link (the source's own page when
// fetched, else a route-prefilled Rome2Rio link — never a generic search) and
// the provenance chain (all sources checked, cheapest first, with the chosen one
// marked). disp is the group price already converted to the display currency.
func (p *PricePlanner) legQuote(from, to string, q conn.TransportQuote, chain []pricefetch.Candidate, tier Tier, disp float64, cur string, payable float64) LegQuote {
	url := q.DeepLink
	if url == "" {
		url = pricefetch.DeepLink(from, to, string(q.Mode), p.tripDate, p.tripAdults)
	}
	alts := make([]LegAlt, 0, len(chain))
	var chosenDetail LegAlt
	for _, c := range chain {
		d, err := p.conv.Convert(c.Price.Amount*payable, c.Price.Currency, cur)
		if err != nil {
			continue
		}
		link := c.URL
		if link == "" {
			link = pricefetch.DeepLink(from, to, string(q.Mode), p.tripDate, p.tripAdults)
		}
		a := LegAlt{
			Source: c.Source, Price: round2(d), URL: link, Chosen: eqFold(c.Source, q.Source),
			Operator: c.Operator, Depart: c.Depart, Duration: c.Duration, Stops: c.Stops, Baggage: c.Baggage,
		}
		if a.Chosen {
			chosenDetail = a
		}
		alts = append(alts, a)
	}
	return LegQuote{
		From: from, To: to, Mode: string(q.Mode),
		Price: round2(disp), Estimated: q.Estimated, Source: q.Source, SourceTier: string(tier),
		URL: url, Alternatives: alts, FetchedAt: fetchedAtStr(q.FetchedAt),
		Operator: chosenDetail.Operator, Depart: chosenDetail.Depart, Duration: chosenDetail.Duration,
		Stops: chosenDetail.Stops, Baggage: chosenDetail.Baggage,
	}
}

// fetchedAtStr formats a provenance timestamp as RFC3339 (empty when zero).
func fetchedAtStr(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// payablePax converts the party into payable-fare units: adults full, children
// discounted (~75%), infants nearly free (~10%). At least 1.
func payablePax(req PlanRequest) float64 {
	a := req.Adults
	if a < 1 {
		a = 1
	}
	p := float64(a) + 0.75*float64(max0(req.Children)) + 0.10*float64(max0(req.Infants))
	if p < 1 {
		p = 1
	}
	return p
}

func max0(n int) int {
	if n < 0 {
		return 0
	}
	return n
}

// isValidLegMode reports whether m is an accepted per-leg override.
func isValidLegMode(m string) bool {
	return m == "cheapest" || m == "flight" || m == "train" || m == "bus" || m == "ferry"
}

// resolveMode maps "cheapest"/"" to a concrete mode for a fare lookup.
func resolveMode(mode string) conn.TransportMode {
	switch mode {
	case "train":
		return conn.ModeTrain
	case "bus":
		return conn.ModeBus
	case "ferry":
		return conn.ModeFerry
	default:
		return conn.ModeFlight
	}
}

// realLeg tries the real fare resolver for a leg, returning an EUR-normalized,
// provenance-tagged quote (Estimated=false) on success.
func (p *PricePlanner) realLeg(ctx context.Context, tenantID, from, to, mode string) (conn.TransportQuote, []pricefetch.Candidate, bool) {
	if p.fares == nil || !p.fares.Enabled() {
		return conn.TransportQuote{}, nil, false
	}
	fc, _ := p.geo.Coords(from)
	tc, _ := p.geo.Coords(to)
	q := pricefetch.LegQuery{
		From: from, To: to,
		FromLat: fc.Lat, FromLng: fc.Lng, ToLat: tc.Lat, ToLng: tc.Lng,
		Mode: resolveMode(mode), DepartDate: p.tripDate, Adults: p.tripAdults,
	}
	if p.fareDeadline > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.fareDeadline)
		defer cancel()
	}
	cand, chain, ok := p.fares.Quote(ctx, q, "EUR", tenantID)
	if !ok {
		return conn.TransportQuote{}, nil, false
	}
	eur, err := p.conv.Convert(cand.Price.Amount, cand.Price.Currency, "EUR")
	if err != nil {
		return conn.TransportQuote{}, nil, false
	}
	return conn.TransportQuote{
		From: from, To: to, Mode: q.Mode,
		Price:      conn.Money{Amount: eur, Currency: "EUR"},
		DeepLink:   cand.URL,
		Provenance: conn.Provenance{Source: cand.Source, FetchedAt: cand.FetchedAt, Estimated: false},
	}, chain, true
}

// cheapestLeg resolves the chosen transport quote for a leg: a real fetched
// price (+ its provenance chain) when available, otherwise the flywheel
// (estimator). "cheapest" picks the min across modes on the estimator path.
func (p *PricePlanner) cheapestLeg(ctx context.Context, tenantID, from, to, mode string) (conn.TransportQuote, []pricefetch.Candidate, Tier, bool) {
	if q, chain, ok := p.realLeg(ctx, tenantID, from, to, mode); ok {
		return q, chain, TierCold, true
	}
	q, tier, ok := p.estimateLeg(ctx, tenantID, from, to, mode)
	return q, nil, tier, ok
}

// estimateLeg resolves a leg using ONLY the local flywheel estimator (no real
// cold fetch). It is the fast path used to build the TSP cost matrix, so routing
// never triggers the O(n^2) network/LLM fetches — those run once per final leg.
func (p *PricePlanner) estimateLeg(ctx context.Context, tenantID, from, to, mode string) (conn.TransportQuote, Tier, bool) {
	// The local estimate must still compute after the overall plan deadline has
	// fired, so detach from a cancelled context (the estimator needs no ctx).
	if ctx.Err() != nil {
		ctx = context.Background()
	}
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

// priceLeg resolves and prices one leg (outbound OR return) consistently: it
// honours a per-leg mode override (else the global mode), falls back to the
// cheapest feasible mode when the requested one is geographically impossible
// (flagging it, C6), and returns the display-currency LegQuote (with detail +
// alternatives + a real deep-link) plus the EUR group amount for the engine.
// ok=false means no mode at all was feasible for the leg.
func (p *PricePlanner) priceLeg(ctx context.Context, tenantID, from, to string, legModes map[string]string, defaultMode string, payable float64, cur string) (lq LegQuote, group conn.Money, estimated, ok bool, err error) {
	legMode := defaultMode
	if m, o := legModes[from+"|"+to]; o && isValidLegMode(m) {
		legMode = m
	}
	q, chain, tier, o := p.cheapestLeg(ctx, tenantID, from, to, legMode)
	unavailable := false
	requested := ""
	if !o && legMode != "" && legMode != "cheapest" {
		requested = legMode
		q, chain, tier, o = p.cheapestLeg(ctx, tenantID, from, to, "cheapest")
		unavailable = o
	}
	if !o {
		return LegQuote{}, conn.Money{}, false, false, nil
	}
	groupEUR := q.Price.Amount * payable // transport scales with payable passengers
	disp, e := p.conv.Convert(groupEUR, q.Price.Currency, cur)
	if e != nil {
		return LegQuote{}, conn.Money{}, false, false, e
	}
	lq = p.legQuote(from, to, q, chain, tier, disp, cur, payable)
	if unavailable {
		lq.RequestedMode, lq.ModeUnavailable = requested, true
	}
	return lq, conn.Money{Amount: groupEUR, Currency: q.Price.Currency}, q.Estimated, true, nil
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
	// Per-call context for the real fare resolver (date + party size).
	p.tripDate, p.tripAdults = req.StartDate, req.Adults
	// Payable-passenger multiplier (children discounted, infants ~free) scales
	// transport + activities; travelers drives the per-person figure.
	payable := payablePax(req)
	travelers := req.Adults + req.Children + req.Infants
	if travelers < 1 {
		travelers = 1
	}

	// Build the node list. When an origin is given it becomes the start and gets
	// NO hotel/nights (it's the departure point); destinations get hotels. The
	// origin is excluded from the destination set even if also listed there.
	origin := strings.TrimSpace(req.Origin)
	if _, ok := p.geo.Coords(origin); origin != "" && !ok {
		origin = "" // origin not geolocatable -> ignore it
	}

	known := make([]string, 0, len(req.Cities)+1)
	if origin != "" {
		known = append(known, origin)
	}
	for _, c := range req.Cities {
		if origin != "" && eqFold(c, origin) {
			continue
		}
		if _, ok := p.geo.Coords(c); ok {
			known = append(known, c)
		}
	}
	if len(known) < 2 {
		return PlanResult{}, ErrPlanTooFewCities
	}

	// isHotelNode reports whether a node name takes a hotel (origin does not).
	isHotelNode := func(name string) bool { return origin == "" || !eqFold(name, origin) }

	startIdx := 0
	if origin == "" {
		startIdx = indexOfFold(known, req.Start)
		if startIdx < 0 {
			startIdx = 0
		}
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
				// Routing uses the fast estimator only; the expensive real fetch
				// runs once per final leg, not for every O(n^2) matrix pair.
				q, _, ok := p.estimateLeg(ctx, tenantID, known[i], known[j], req.Mode)
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
		lq, grp, est, ok, err := p.priceLeg(ctx, tenantID, order[i], order[i+1], req.LegModes, req.Mode, payable, cur)
		if err != nil {
			return PlanResult{}, err
		}
		if !ok {
			return PlanResult{}, ErrLegUnavailable
		}
		if est {
			anyEstimated = true
		}
		legs = append(legs, lq)
		priced = append(priced, planner.PricedLeg{From: i, To: i + 1, Mode: conn.TransportMode(lq.Mode), Price: grp})
	}

	// Round-trip: price the return leg (last -> origin/start) for the SAME selected
	// mode (or per-leg override), with the same unavailable-mode fallback, its own
	// detail/alternatives/real link — just like an outbound leg (D13). The origin
	// still takes no hotel/nights (C3). Included in the total.
	if variant == string(planner.RoundTrip) && len(order) >= 2 {
		from, to := order[len(order)-1], order[0]
		if lq, grp, est, ok, err := p.priceLeg(ctx, tenantID, from, to, req.LegModes, req.Mode, payable, cur); err == nil && ok {
			if est {
				anyEstimated = true
			}
			legs = append(legs, lq)
			priced = append(priced, planner.PricedLeg{From: len(order) - 1, To: 0, Mode: conn.TransportMode(lq.Mode), Price: grp})
		}
	}

	// Nights are driven by the dates (totalNights = return - depart) and split
	// across the DESTINATIONS only (the origin never takes a hotel/nights).
	var hotelIdx []int
	for i, name := range order {
		if isHotelNode(name) {
			hotelIdx = append(hotelIdx, i)
		}
	}
	destNights, totalNights := allocateNightsN(len(hotelIdx), req, nights)
	// Per-city override (C8): an explicit night count from the stepper wins over
	// the allocation; the trip total is recomputed from the applied values.
	if len(req.CityNights) > 0 {
		lc := make(map[string]int, len(req.CityNights))
		for name, n := range req.CityNights {
			lc[strings.ToLower(strings.TrimSpace(name))] = n
		}
		total := 0
		for k, oi := range hotelIdx {
			if n, ok := lc[strings.ToLower(strings.TrimSpace(order[oi]))]; ok && n > 0 {
				destNights[k] = n
			}
			total += destNights[k]
		}
		totalNights = total
	}
	nightsByCity := make([]int, len(order)) // origin stays 0
	for k, oi := range hotelIdx {
		nightsByCity[oi] = destNights[k]
	}

	// Hotels + activities via the estimator (destinations only).
	stars := tierStars[req.Tier]
	if stars == 0 {
		stars = 4
	}
	// Per-city star override (D6), keyed by lower-cased city name (clamped 2..5).
	cityStars := make(map[string]int, len(req.CityStars))
	for name, s := range req.CityStars {
		if s >= 2 && s <= 5 {
			cityStars[strings.ToLower(strings.TrimSpace(name))] = s
		}
	}
	hotels := make([]planner.PricedHotel, 0, len(hotelIdx))
	hotelLines := make([]HotelLine, 0, len(hotelIdx))
	for _, oi := range hotelIdx {
		city := order[oi]
		cityStar := stars
		if s, ok := cityStars[strings.ToLower(strings.TrimSpace(city))]; ok {
			cityStar = s
		}
		h, err := p.hotels.Hotel(ctx, city, cityStar)
		if err != nil {
			return PlanResult{}, err
		}
		if h.Estimated {
			anyEstimated = true
		}
		nights := nightsByCity[oi]
		hotels = append(hotels, planner.PricedHotel{City: oi, Stars: h.Stars, Nights: nights, Nightly: h.Nightly})
		nightlyDisp, _ := p.conv.Convert(h.Nightly.Amount, h.Nightly.Currency, cur)
		// Alternatives: the other star tiers for this city (cheapest first), so a
		// single city's hotel can be switched and re-priced live (D6).
		var alts []HotelAlt
		for _, t := range []int{3, 4, 5} {
			ah, err := p.hotels.Hotel(ctx, city, t)
			if err != nil {
				continue
			}
			an, _ := p.conv.Convert(ah.Nightly.Amount, ah.Nightly.Currency, cur)
			alts = append(alts, HotelAlt{
				Stars: ah.Stars, Nightly: round2(an), Total: round2(an * float64(nights)),
				Chosen: ah.Stars == h.Stars,
			})
		}
		hotelLines = append(hotelLines, HotelLine{
			City: city, Stars: h.Stars, Name: h.Name, Nights: nights,
			Nightly: round2(nightlyDisp), Total: round2(nightlyDisp * float64(nights)),
			Estimated: h.Estimated, Source: h.Source, FetchedAt: fetchedAtStr(h.FetchedAt),
			URL:  pricefetch.HotelLink(city, req.StartDate, req.ReturnDate, req.Adults),
			Alts: alts,
		})
	}
	activitiesEUR := activitiesPerNightEUR * float64(totalNights) * payable
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
		Order: order, Variant: variant, Legs: legs, HotelLines: hotelLines,
		Transport: bd.Transport, Hotels: bd.Hotels, Activities: bd.Activities, Total: bd.Total,
		Currency: cur, HotelStars: stars, Nights: totalNights, NightsByCity: nightsByCity,
		Travelers: travelers, PerPerson: round2(bd.Total / float64(travelers)),
		ActivitiesPerNight: round2(actPerNightDisp), Estimated: anyEstimated,
	}
	if altVariant != "" {
		if c, err := p.conv.Convert(altCostEUR, "EUR", cur); err == nil {
			// The alternative shares the city sequence/nights (it differs by
			// variant: round-trip return leg vs open-jaw), so surface its full
			// order + night split + total for the UI (C5).
			res.AltVariant, res.AltCost = altVariant, round2(c)
			res.AltOrder = append([]string(nil), order...)
			res.AltNightsByCity = append([]int(nil), nightsByCity...)
		}
	}
	return res, nil
}

// allocateNightsN derives per-destination nights for n destinations from the
// trip dates (AllocateNights over total = return - depart); without valid dates
// it uses fallbackPer each.
func allocateNightsN(n int, req PlanRequest, fallbackPer int) ([]int, int) {
	if n <= 0 {
		return nil, 0
	}
	if d := nightsBetween(req.StartDate, req.ReturnDate); d > 0 {
		return planner.AllocateNights(d, n, 1, nil), d
	}
	by := make([]int, n)
	for i := range by {
		by[i] = fallbackPer
	}
	return by, fallbackPer * n
}

// nightsBetween returns the integer nights between two YYYY-MM-DD dates (0 if
// unparseable or non-positive).
func nightsBetween(start, end string) int {
	const f = "2006-01-02"
	s, err1 := time.Parse(f, start)
	e, err2 := time.Parse(f, end)
	if err1 != nil || err2 != nil {
		return 0
	}
	d := int(e.Sub(s).Hours() / 24)
	if d < 0 {
		return 0
	}
	return d
}

func eqFold(a, b string) bool { return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b)) }

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
