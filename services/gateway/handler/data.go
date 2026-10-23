package handler

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/cache"
	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/httpx"
	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/tenant"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/ai"
	conn "github.com/KhanbalaRashidov/ai-travel-platform/services/connectors"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/connectors/estimate"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/connectors/places"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/dataflow"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/knowledge"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/planner"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/pricefetch"
)

// DataHandler serves flywheel-backed quotes, dynamic destinations/places, POIs
// and the fully priced plan. It holds the pricing building blocks so it can
// build a request-scoped planner whose geocoder knows the exact cities the user
// picked via autocomplete (coordinates supplied in the request).
type DataHandler struct {
	routes  *dataflow.RouteFlywheel
	poi     conn.POISource
	store   knowledge.Store
	kv      cache.KV
	conv    planner.Converter
	seedGeo conn.Geocoder
	places  *places.Client
	fares   *pricefetch.Resolver
	llm     ai.LLM // for minimal-mode destination suggestions (may be nil)
}

// NewDataHandler wires the data handler.
func NewDataHandler(routes *dataflow.RouteFlywheel, poi conn.POISource, store knowledge.Store, kv cache.KV, conv planner.Converter, seedGeo conn.Geocoder, pl *places.Client, fares *pricefetch.Resolver, llm ai.LLM) *DataHandler {
	return &DataHandler{routes: routes, poi: poi, store: store, kv: kv, conv: conv, seedGeo: seedGeo, places: pl, fares: fares, llm: llm}
}

// Plan handles POST /data/plan — a fully priced plan computed server-side: the
// route (TSP), per-leg transport quotes (flywheel), hotels + activities
// (estimator) and FX, all in one response with provenance/estimated flags. This
// is the single source of truth for manual planning, tier/mode re-price and the
// chat refine loop. Supplying `order` re-prices that order without re-sequencing.
// The request's places (with coordinates) make any selected city priceable.
func (h *DataHandler) Plan(c *fiber.Ctx) error {
	var req dataflow.PlanRequest
	if err := c.BodyParser(&req); err != nil {
		return httpx.Err(c, fiber.StatusBadRequest, "invalid request")
	}

	// Minimal mode: only an origin (+ budget) was given — ask the LLM to suggest
	// destinations, then geolocate them. The engine still prices everything.
	if strings.TrimSpace(req.Origin) != "" && len(req.Cities) == 0 {
		if h.llm == nil {
			return httpx.Err(c, fiber.StatusServiceUnavailable, "AI not configured (set the LLM key) for destination suggestions")
		}
		names, err := ai.SuggestDestinations(c.Context(), h.llm, req.Origin, req.BudgetEUR, 5)
		if err != nil || len(names) == 0 {
			return httpx.Err(c, fiber.StatusBadGateway, "could not suggest destinations")
		}
		for _, n := range names {
			res, err := h.places.Search(c.Context(), n, "en", 1)
			if err == nil && len(res) > 0 {
				req.Cities = append(req.Cities, res[0].Name)
				req.Places = append(req.Places, dataflow.PlaceCoord{Name: res[0].Name, Lat: res[0].Lat, Lng: res[0].Lng})
			}
		}
		if len(req.Cities) == 0 {
			return httpx.Err(c, fiber.StatusBadGateway, "could not resolve suggested destinations")
		}
	}

	if len(req.Cities) < 1 || (strings.TrimSpace(req.Origin) == "" && len(req.Cities) < 2) {
		return httpx.Err(c, fiber.StatusBadRequest, "need an origin + destination, or at least 2 cities")
	}
	// Request-scoped geocoder (client coords first, then the seed) + estimator +
	// flywheel, sharing the durable store/cache so results still cache.
	geo := conn.MergedGeocoder{req.Geocoder(), h.seedGeo}
	est := estimate.New(geo, 0)
	fw := dataflow.NewRouteFlywheel(h.kv, h.store, est, 0)
	pricing := dataflow.NewPricePlanner(fw, est, geo, h.conv)
	if h.fares != nil {
		// Cap the real (cold) fetch per leg so a plan always returns promptly;
		// cache hits are instant, a slow source falls back to the estimate (§6).
		pricing = pricing.WithFares(h.fares).WithFareDeadline(4 * time.Second)
	}

	// Hard overall deadline so a plan NEVER blocks on slow real-price fetches:
	// when it fires, in-flight source/LLM calls are cancelled and every leg that
	// hasn't resolved falls back to the labeled estimate — the plan still returns.
	ctx, cancel := context.WithTimeout(c.Context(), planDeadline)
	defer cancel()

	res, err := pricing.Plan(ctx, tenant.FromContext(c), req)
	if err != nil {
		if errors.Is(err, dataflow.ErrPlanTooFewCities) {
			return httpx.Err(c, fiber.StatusBadRequest, "at least 2 known cities are required")
		}
		if errors.Is(err, dataflow.ErrLegUnavailable) {
			return httpx.Err(c, fiber.StatusBadGateway, "no transport available for part of this route")
		}
		return httpx.Err(c, fiber.StatusInternalServerError, "could not price the plan")
	}
	return httpx.OK(c, res)
}

// planDeadline bounds total /data/plan time so the client never waits on slow
// cold fetches; past it, unresolved legs use the labeled estimate.
const planDeadline = 9 * time.Second

var validModes = map[string]conn.TransportMode{
	"flight": conn.ModeFlight,
	"train":  conn.ModeTrain,
	"bus":    conn.ModeBus,
}

// Transport handles GET /data/transport?from=&to=&mode= — a single leg quote,
// resolved through the hot -> warm -> cold flywheel. The response reports which
// tier served it so the UI can show freshness/provenance.
func (h *DataHandler) Transport(c *fiber.Ctx) error {
	from := strings.TrimSpace(c.Query("from"))
	to := strings.TrimSpace(c.Query("to"))
	if from == "" || to == "" {
		return httpx.Err(c, fiber.StatusBadRequest, "from and to are required")
	}
	modeStr := strings.ToLower(strings.TrimSpace(c.Query("mode")))
	if modeStr == "" {
		modeStr = "flight"
	}
	mode, ok := validModes[modeStr]
	if !ok {
		return httpx.Err(c, fiber.StatusBadRequest, "mode must be flight, train or bus")
	}

	q, tier, err := h.routes.Route(c.Context(), tenant.FromContext(c), from, to, mode)
	if err != nil {
		if errors.Is(err, conn.ErrUnavailable) {
			return httpx.Err(c, fiber.StatusBadGateway, "no transport quote available for this leg")
		}
		return httpx.Err(c, fiber.StatusInternalServerError, "could not resolve transport quote")
	}
	return httpx.OK(c, fiber.Map{"quote": q, "source_tier": string(tier)})
}

// Destinations handles GET /data/destinations — the dynamic, knowledge-driven
// city list (replaces hardcoded chips). Grows as trips are learned.
func (h *DataHandler) Destinations(c *fiber.Ctx) error {
	ds, err := h.store.ListDestinations(c.Context(), tenant.FromContext(c))
	if err != nil {
		return httpx.Err(c, fiber.StatusInternalServerError, "could not load destinations")
	}
	return httpx.OK(c, ds)
}

// POIs handles GET /data/poi?city=&limit= — points of interest for a city from
// the POI connector (OSM Overpass in production).
func (h *DataHandler) POIs(c *fiber.Ctx) error {
	city := strings.TrimSpace(c.Query("city"))
	if city == "" {
		return httpx.Err(c, fiber.StatusBadRequest, "city is required")
	}
	limit := 10
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	// Primary: open data (OSM Overpass). Many cities return nothing, so fall back
	// to the model's knowledge of popular places (C9) — the list is rarely empty.
	pois, err := h.poi.POIs(c.Context(), city, limit)
	if err != nil && !errors.Is(err, conn.ErrUnavailable) {
		return httpx.Err(c, fiber.StatusInternalServerError, "could not load points of interest")
	}
	if len(pois) < limit && h.llm != nil {
		pois = h.fillPOIsFromAI(c.Context(), city, limit, pois, c.Query("locale"))
	}
	return httpx.OK(c, pois)
}

// fillPOIsFromAI appends knowledge-based popular places (deduped by name) until
// the list reaches limit. AI POIs carry Source="ai-knowledge", Estimated prices,
// and NO rating (ratings are never invented).
func (h *DataHandler) fillPOIsFromAI(ctx context.Context, city string, limit int, have []conn.POI, locale string) []conn.POI {
	infos, err := ai.PopularPOIs(ctx, h.llm, city, langName(locale), limit)
	if err != nil {
		return have
	}
	seen := make(map[string]bool, len(have))
	for _, p := range have {
		seen[strings.ToLower(strings.TrimSpace(p.Name))] = true
	}
	now := time.Now().UTC()
	for _, in := range infos {
		if in.Name == "" || seen[strings.ToLower(in.Name)] {
			continue
		}
		seen[strings.ToLower(in.Name)] = true
		have = append(have, conn.POI{
			City: city, Name: in.Name, Category: in.Category,
			Desc: in.Desc, Free: in.Free, Price: strings.TrimSpace(in.Price),
			Provenance: conn.Provenance{Source: "ai-knowledge", FetchedAt: now, Estimated: true},
		})
		if len(have) >= limit {
			break
		}
	}
	return have
}

// Places handles GET /data/places?q= — city/airport autocomplete from the
// Travelpayouts places endpoint, cached in the hot layer (short TTL). Returns
// geolocated results so the client can plan any selected city.
func (h *DataHandler) Places(c *fiber.Ctx) error {
	q := strings.TrimSpace(c.Query("q"))
	if len(q) < 2 {
		return httpx.OK(c, []places.Place{})
	}
	locale := strings.ToLower(strings.TrimSpace(c.Query("locale")))
	if locale == "" {
		locale = "en"
	}
	key := "places|" + locale + "|" + strings.ToLower(q)
	if cached, err := h.kv.Get(c.Context(), key); err == nil && cached != "" {
		c.Set("Content-Type", "application/json")
		return c.SendString(`{"success":true,"data":` + cached + `}`)
	}
	res, err := h.places.Search(c.Context(), q, locale, 8)
	if err != nil {
		if errors.Is(err, conn.ErrUnavailable) {
			return httpx.Err(c, fiber.StatusBadGateway, "autocomplete is unavailable")
		}
		return httpx.Err(c, fiber.StatusInternalServerError, "autocomplete failed")
	}
	if b, err := json.Marshal(res); err == nil {
		_ = h.kv.Set(c.Context(), key, string(b), 6*time.Hour)
	}
	return httpx.OK(c, res)
}
