package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/httpx"
	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/tenant"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/knowledge"
)

// RecommendHandler exposes the knowledge base's popularity-driven suggestions.
type RecommendHandler struct {
	svc *knowledge.Service
}

// NewRecommendHandler builds the handler.
func NewRecommendHandler(svc *knowledge.Service) *RecommendHandler {
	return &RecommendHandler{svc: svc}
}

// List handles GET /recommendations?limit=5 — top destinations for the tenant.
func (h *RecommendHandler) List(c *fiber.Ctx) error {
	limit := 5
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	recs, err := h.svc.Suggest(c.Context(), tenant.FromContext(c), limit)
	if err != nil {
		return httpx.Err(c, fiber.StatusInternalServerError, "could not load recommendations")
	}
	return httpx.OK(c, recs)
}

type visitReq struct {
	Destination string `json:"destination"`
	Kind        string `json:"kind"`
}

// Visit handles POST /recommendations/visit — record a destination choice so
// the knowledge base and ranking grow (the data flywheel).
func (h *RecommendHandler) Visit(c *fiber.Ctx) error {
	var req visitReq
	if err := c.BodyParser(&req); err != nil || req.Destination == "" {
		return httpx.Err(c, fiber.StatusBadRequest, "destination is required")
	}
	if err := h.svc.RecordVisit(c.Context(), tenant.FromContext(c), req.Destination, req.Kind); err != nil {
		return httpx.Err(c, fiber.StatusInternalServerError, "could not record visit")
	}
	return httpx.OK(c, fiber.Map{"recorded": req.Destination})
}
