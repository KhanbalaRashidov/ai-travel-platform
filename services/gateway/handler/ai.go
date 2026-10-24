package handler

import (
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/httpx"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/ai"
)

// busyMsg is returned when the provider is momentarily overloaded/rate-limited
// (transient 429/503); the client shows a "try again shortly" state.
const busyMsg = "AI service is busy, try again shortly"

// AIHandler serves the LLM features: the itinerary writer (bulk model), chat
// (reasoning model + guardrail, with planner/flywheel tools), and the
// conversational refine loop. When the Anthropic key is unset both LLMs are nil
// and the endpoints report 503 "AI not configured" — the rest of the platform
// is unaffected.
type AIHandler struct {
	primary ai.LLM // reasoning (chat + edit parsing)
	bulk    ai.LLM // cheap formatting (itinerary)
	conv    currencyConverter
	tools   []ai.Tool
}

// currencyConverter normalizes amounts between currencies (planner.StaticConverter
// satisfies it); used to convert the roadmap's EUR food estimate to the display
// currency (D4).
type currencyConverter interface {
	Convert(amount float64, from, to string) (float64, error)
}

// NewAIHandler wires the handler. Pass nil LLMs when the key is not configured;
// tools (planner/flywheel) are attached to the chat orchestrator. conv may be nil
// (food estimates then stay in EUR).
func NewAIHandler(primary, bulk ai.LLM, conv currencyConverter, tools ...ai.Tool) *AIHandler {
	return &AIHandler{primary: primary, bulk: bulk, conv: conv, tools: tools}
}

// Enabled reports whether the AI features are configured.
func (h *AIHandler) Enabled() bool { return h.primary != nil && h.bulk != nil }

type legFactDTO struct {
	From  string  `json:"from"`
	To    string  `json:"to"`
	Mode  string  `json:"mode"`
	Price float64 `json:"price"`
}

type itineraryReq struct {
	Cities          []string     `json:"cities"`
	Legs            []legFactDTO `json:"legs"`
	Nights          int          `json:"nights"`
	NightsByCity    []int        `json:"nights_by_city"`
	Currency        string       `json:"currency"`
	TransportTotal  float64      `json:"transport_total"`
	HotelsTotal     float64      `json:"hotels_total"`
	ActivitiesTotal float64      `json:"activities_total"`
	Total           float64      `json:"total"`
	Locale          string       `json:"locale"`
}

// langName maps a UI locale to the language the LLM should reply in.
func langName(locale string) string {
	switch strings.ToLower(strings.TrimSpace(locale)) {
	case "az":
		return "Azerbaijani"
	case "ru":
		return "Russian"
	case "tr":
		return "Turkish"
	default:
		return "English"
	}
}

// Itinerary handles POST /ai/itinerary — turn a priced plan into prose. The LLM
// may not invent prices: any ungrounded amount is surfaced as a violation.
func (h *AIHandler) Itinerary(c *fiber.Ctx) error {
	if h.bulk == nil {
		return httpx.Err(c, fiber.StatusServiceUnavailable, "AI not configured")
	}
	var req itineraryReq
	if err := c.BodyParser(&req); err != nil || len(req.Cities) == 0 {
		return httpx.Err(c, fiber.StatusBadRequest, "cities are required")
	}
	legs := make([]ai.LegFact, len(req.Legs))
	for i, l := range req.Legs {
		legs[i] = ai.LegFact{From: l.From, To: l.To, Mode: l.Mode, Price: l.Price}
	}
	brief := ai.PlanBrief{
		Cities: req.Cities, Legs: legs, Nights: req.Nights, NightsByCity: req.NightsByCity,
		Language: langName(req.Locale), Currency: req.Currency,
		TransportTotal: req.TransportTotal, HotelsTotal: req.HotelsTotal,
		ActivitiesTotal: req.ActivitiesTotal, Total: req.Total,
	}
	res, err := ai.WriteItinerary(c.Context(), h.bulk, brief, 1)
	if err != nil {
		if ai.IsTransient(err) {
			return httpx.Err(c, fiber.StatusServiceUnavailable, busyMsg)
		}
		return httpx.Err(c, fiber.StatusBadGateway, "itinerary generation failed")
	}
	return httpx.OK(c, res)
}

type placeCoordDTO struct {
	Name string  `json:"name"`
	Lat  float64 `json:"lat"`
	Lng  float64 `json:"lng"`
}

type roadmapReq struct {
	Cities       []string        `json:"cities"`
	NightsByCity []int           `json:"nights_by_city"`
	Places       []placeCoordDTO `json:"places"`
	Locale       string          `json:"locale"`
	Currency     string          `json:"currency"`
}

// Roadmap handles POST /ai/roadmap — a STRUCTURED, day-by-day, location-based
// itinerary (per-city summary + season, day POIs with coords and free/paid
// prices) the client renders as day cards + map pins, never raw markdown.
// Transport/hotel prices are never invented here (guardrail); activity prices
// are the model's approximate, "~"-marked estimates.
func (h *AIHandler) Roadmap(c *fiber.Ctx) error {
	if h.bulk == nil {
		return httpx.Err(c, fiber.StatusServiceUnavailable, "AI not configured")
	}
	var req roadmapReq
	if err := c.BodyParser(&req); err != nil || len(req.Cities) == 0 {
		return httpx.Err(c, fiber.StatusBadRequest, "cities are required")
	}
	coords := make(map[string]ai.LatLng, len(req.Places))
	for _, p := range req.Places {
		if p.Lat != 0 || p.Lng != 0 {
			coords[strings.ToLower(strings.TrimSpace(p.Name))] = ai.LatLng{Lat: p.Lat, Lng: p.Lng}
		}
	}
	// Never 502 the roadmap: on any LLM timeout/error/parse failure, serve the
	// deterministic skeleton (days + cities + nights) and let POIs fill lazily.
	res, err := ai.StructuredRoadmap(c.Context(), h.bulk, req.Cities, req.NightsByCity, coords, langName(req.Locale))
	if err != nil {
		log.Printf("ai roadmap: degraded to skeleton: %v", err)
	}
	// Convert the EUR food estimate (D4) to the display currency, so the live
	// cost stays single-currency. Best-effort: leave EUR if conversion fails.
	cur := strings.ToUpper(strings.TrimSpace(req.Currency))
	if h.conv != nil && cur != "" && cur != "EUR" {
		for i := range res.Food {
			if v, e := h.conv.Convert(res.Food[i].PerDay, "EUR", cur); e == nil {
				res.Food[i].PerDay = round2(v)
				res.Food[i].Total = round2(v * float64(res.Food[i].Nights))
			}
		}
	}
	return httpx.OK(c, res)
}

// round2 rounds to 2 decimals.
func round2(f float64) float64 { return float64(int64(f*100+0.5)) / 100 }

type destinationsReq struct {
	Countries []string `json:"countries"`
	BudgetEUR float64  `json:"budget_eur"`
	Currency  string   `json:"currency"`
	Locale    string   `json:"locale"`
}

// Destinations handles POST /ai/destinations — country mode (D12): given one or
// more countries (+ optional budget), return AI-ranked city suggestions per
// country (reason + best season) and optional budget guidance. Ratings are never
// invented (stripped in the AI layer). 503 when AI is not configured.
func (h *AIHandler) Destinations(c *fiber.Ctx) error {
	if h.bulk == nil {
		return httpx.Err(c, fiber.StatusServiceUnavailable, "AI not configured")
	}
	var req destinationsReq
	if err := c.BodyParser(&req); err != nil || len(req.Countries) == 0 {
		return httpx.Err(c, fiber.StatusBadRequest, "countries are required")
	}
	res, err := ai.SuggestCountryDestinations(c.Context(), h.bulk, req.Countries, req.BudgetEUR, req.Currency, langName(req.Locale))
	if err != nil {
		if ai.IsTransient(err) {
			return httpx.Err(c, fiber.StatusServiceUnavailable, busyMsg)
		}
		return httpx.Err(c, fiber.StatusBadGateway, "could not suggest destinations")
	}
	return httpx.OK(c, res)
}

type chatReq struct {
	Message string `json:"message"`
	Locale  string `json:"locale"`
}

const chatSystem = "You are a helpful, concise travel assistant for an advisory-only planner " +
	"(no booking, no payment, no visa or legal advice). Answer briefly and practically. " +
	"Do NOT quote specific prices; point users to the planner for costs."

// Chat handles POST /ai/chat — a short travel Q&A turn. The price guardrail runs
// with an empty grounding set, so any price the model emits is flagged.
func (h *AIHandler) Chat(c *fiber.Ctx) error {
	if h.primary == nil {
		return httpx.Err(c, fiber.StatusServiceUnavailable, "AI not configured")
	}
	var req chatReq
	if err := c.BodyParser(&req); err != nil || req.Message == "" {
		return httpx.Err(c, fiber.StatusBadRequest, "message is required")
	}
	orch := ai.NewOrchestrator(h.primary, 4, 1, h.tools...)
	res, err := orch.Run(c.Context(), chatSystem+" Reply in "+langName(req.Locale)+".", req.Message)
	if err != nil {
		if ai.IsTransient(err) {
			return httpx.Err(c, fiber.StatusServiceUnavailable, busyMsg)
		}
		return httpx.Err(c, fiber.StatusBadGateway, "chat failed")
	}
	return httpx.OK(c, res)
}

type refineReq struct {
	Message string       `json:"message"`
	State   ai.EditState `json:"state"`
}

type refineResp struct {
	State   ai.EditState `json:"state"`
	Changed []string     `json:"changed"`
}

// Refine handles POST /ai/refine — the conversational edit loop. The LLM turns a
// request like "make it cheaper" / "drop Istanbul" / "add a day" into a
// structured TripEdit, which is applied deterministically. The client then
// re-runs the TSP + re-pricing engine with the returned state, so the itinerary
// and cost panel update live. The LLM never computes routes or prices.
func (h *AIHandler) Refine(c *fiber.Ctx) error {
	if h.primary == nil {
		return httpx.Err(c, fiber.StatusServiceUnavailable, "AI not configured")
	}
	var req refineReq
	if err := c.BodyParser(&req); err != nil || strings.TrimSpace(req.Message) == "" {
		return httpx.Err(c, fiber.StatusBadRequest, "message is required")
	}
	started := time.Now()
	edit, err := ai.ExtractEdit(c.Context(), h.primary, req.Message, summarizeState(req.State))
	elapsed := time.Since(started)
	if err != nil {
		// The error carries a masked/raw snippet of the model reply for diagnosis.
		log.Printf("ai refine: failed in %s: %v", elapsed.Round(time.Millisecond), err)
		if ai.IsTransient(err) {
			return httpx.Err(c, fiber.StatusServiceUnavailable, busyMsg)
		}
		return httpx.Err(c, fiber.StatusBadGateway, "could not understand the edit")
	}
	log.Printf("ai refine: ok in %s", elapsed.Round(time.Millisecond))
	newState, changed := ai.ApplyEdit(req.State, edit)
	return httpx.OK(c, refineResp{State: newState, Changed: changed})
}

// summarizeState renders the current wizard state for the edit-parsing prompt.
func summarizeState(s ai.EditState) string {
	var b strings.Builder
	b.WriteString("cities: " + strings.Join(s.Cities, ", "))
	b.WriteString("; start: " + s.Start)
	b.WriteString("; tier: " + s.Tier)
	b.WriteString("; mode: " + s.Mode)
	b.WriteString("; nights: ")
	b.WriteString(strconv.Itoa(s.Nights))
	return b.String()
}
