package handler

import (
	"github.com/gofiber/fiber/v2"

	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/httpx"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/ai"
)

// AIHandler serves the LLM features: the itinerary writer (bulk model) and chat
// (reasoning model + guardrail). When the Anthropic key is unset both LLMs are
// nil and the endpoints report 503 "AI not configured" — the rest of the
// platform is unaffected.
type AIHandler struct {
	primary ai.LLM // reasoning (chat)
	bulk    ai.LLM // cheap formatting (itinerary)
}

// NewAIHandler wires the handler. Pass nil LLMs when the key is not configured.
func NewAIHandler(primary, bulk ai.LLM) *AIHandler {
	return &AIHandler{primary: primary, bulk: bulk}
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
	Currency        string       `json:"currency"`
	TransportTotal  float64      `json:"transport_total"`
	HotelsTotal     float64      `json:"hotels_total"`
	ActivitiesTotal float64      `json:"activities_total"`
	Total           float64      `json:"total"`
}

// Itinerary handles POST /ai/itinerary — turn a priced plan into prose. The LLM
// may not invent prices: any ungrounded amount is surfaced as a violation.
func (h *AIHandler) Itinerary(c *fiber.Ctx) error {
	if h.bulk == nil {
		return httpx.Err(c, fiber.StatusServiceUnavailable, "AI not configured (set ANTHROPIC_API_KEY)")
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
		Cities: req.Cities, Legs: legs, Nights: req.Nights, Currency: req.Currency,
		TransportTotal: req.TransportTotal, HotelsTotal: req.HotelsTotal,
		ActivitiesTotal: req.ActivitiesTotal, Total: req.Total,
	}
	res, err := ai.WriteItinerary(c.Context(), h.bulk, brief, 1)
	if err != nil {
		return httpx.Err(c, fiber.StatusBadGateway, "itinerary generation failed")
	}
	return httpx.OK(c, res)
}

type chatReq struct {
	Message string `json:"message"`
}

const chatSystem = "You are a helpful, concise travel assistant for an advisory-only planner " +
	"(no booking, no payment, no visa or legal advice). Answer briefly and practically. " +
	"Do NOT quote specific prices; point users to the planner for costs."

// Chat handles POST /ai/chat — a short travel Q&A turn. The price guardrail runs
// with an empty grounding set, so any price the model emits is flagged.
func (h *AIHandler) Chat(c *fiber.Ctx) error {
	if h.primary == nil {
		return httpx.Err(c, fiber.StatusServiceUnavailable, "AI not configured (set ANTHROPIC_API_KEY)")
	}
	var req chatReq
	if err := c.BodyParser(&req); err != nil || req.Message == "" {
		return httpx.Err(c, fiber.StatusBadRequest, "message is required")
	}
	orch := ai.NewOrchestrator(h.primary, 4, 1)
	res, err := orch.Run(c.Context(), chatSystem, req.Message)
	if err != nil {
		return httpx.Err(c, fiber.StatusBadGateway, "chat failed")
	}
	return httpx.OK(c, res)
}
