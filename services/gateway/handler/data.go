package handler

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/httpx"
	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/tenant"
	conn "github.com/KhanbalaRashidov/ai-travel-platform/services/connectors"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/dataflow"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/knowledge"
)

// DataHandler serves flywheel-backed quotes and knowledge-backed destinations.
type DataHandler struct {
	routes  *dataflow.RouteFlywheel
	poi     conn.POISource
	store   knowledge.Store
	pricing *dataflow.PricePlanner
}

// NewDataHandler wires the data handler.
func NewDataHandler(routes *dataflow.RouteFlywheel, poi conn.POISource, store knowledge.Store, pricing *dataflow.PricePlanner) *DataHandler {
	return &DataHandler{routes: routes, poi: poi, store: store, pricing: pricing}
}

// Plan handles POST /data/plan — a fully priced plan computed server-side: the
// route (TSP), per-leg transport quotes (flywheel), hotels + activities
// (estimator) and FX, all in one response with provenance/estimated flags. This
// is the single source of truth for manual planning, tier/mode re-price and the
// chat refine loop. Supplying `order` re-prices that order without re-sequencing.
func (h *DataHandler) Plan(c *fiber.Ctx) error {
	var req dataflow.PlanRequest
	if err := c.BodyParser(&req); err != nil || len(req.Cities) < 2 {
		return httpx.Err(c, fiber.StatusBadRequest, "at least 2 cities are required")
	}
	res, err := h.pricing.Plan(c.Context(), tenant.FromContext(c), req)
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
	pois, err := h.poi.POIs(c.Context(), city, limit)
	if err != nil {
		if errors.Is(err, conn.ErrUnavailable) {
			return httpx.Err(c, fiber.StatusBadGateway, "points of interest unavailable for this city")
		}
		return httpx.Err(c, fiber.StatusInternalServerError, "could not load points of interest")
	}
	return httpx.OK(c, pois)
}
