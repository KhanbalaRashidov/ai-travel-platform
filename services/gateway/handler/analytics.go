package handler

import (
	"sync"

	"github.com/gofiber/fiber/v2"

	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/httpx"
)

// allowedEvents is a fixed allowlist, so analytics can only ever record these
// anonymous, aggregate counters — no free-form names, no PII, no trip contents.
var allowedEvents = map[string]bool{
	"plan_computed":        true,
	"itinerary_generated":  true,
	"chat_message":         true,
	"refine_applied":       true,
	"place_searched":       true,
	"recommendations_view": true,
}

// AnalyticsHandler keeps privacy-safe aggregate event counts in memory. It
// stores only counts of allowlisted event names — nothing that identifies a
// user or a trip. (Durable storage is an optional follow-up.)
type AnalyticsHandler struct {
	mu     sync.Mutex
	counts map[string]int64
}

// NewAnalyticsHandler builds an empty analytics counter.
func NewAnalyticsHandler() *AnalyticsHandler {
	return &AnalyticsHandler{counts: make(map[string]int64)}
}

type eventReq struct {
	Name string `json:"name"`
}

// Event handles POST /analytics/event — increments one allowlisted counter.
func (h *AnalyticsHandler) Event(c *fiber.Ctx) error {
	var req eventReq
	if err := c.BodyParser(&req); err != nil || !allowedEvents[req.Name] {
		return httpx.Err(c, fiber.StatusBadRequest, "unknown event")
	}
	h.mu.Lock()
	h.counts[req.Name]++
	h.mu.Unlock()
	return httpx.OK(c, fiber.Map{"recorded": req.Name})
}

// Stats handles GET /analytics/stats — the aggregate counts (no PII).
func (h *AnalyticsHandler) Stats(c *fiber.Ctx) error {
	h.mu.Lock()
	out := make(map[string]int64, len(h.counts))
	for k, v := range h.counts {
		out[k] = v
	}
	h.mu.Unlock()
	return httpx.OK(c, out)
}
