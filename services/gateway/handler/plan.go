package handler

import (
	"github.com/gofiber/fiber/v2"

	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/httpx"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/gateway/service"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/planner"
)

// PlanHandler exposes the cost-optimization engine over HTTP.
type PlanHandler struct {
	svc *service.PlanService
}

// NewPlanHandler builds a PlanHandler.
func NewPlanHandler(svc *service.PlanService) *PlanHandler {
	return &PlanHandler{svc: svc}
}

type sequenceReq struct {
	Matrix planner.Matrix `json:"matrix"`
	Start  int            `json:"start"`
	End    *int           `json:"end"` // nil/omitted => free end (-1)
}

type routeDTO struct {
	Variant string  `json:"variant"`
	Order   []int   `json:"order"`
	Cost    float64 `json:"cost"`
}

func toRouteDTO(r planner.Route) routeDTO {
	return routeDTO{Variant: string(r.Variant), Order: r.Order, Cost: r.Cost}
}

// Sequence handles POST /plan/sequence: order cities to minimize transport cost.
func (h *PlanHandler) Sequence(c *fiber.Ctx) error {
	var req sequenceReq
	if err := c.BodyParser(&req); err != nil {
		return httpx.Err(c, fiber.StatusBadRequest, "invalid request body")
	}
	if len(req.Matrix) == 0 {
		return httpx.Err(c, fiber.StatusBadRequest, "matrix is required")
	}
	end := -1
	if req.End != nil {
		end = *req.End
	}
	best, alt, err := h.svc.Sequence(req.Matrix, req.Start, end)
	if err != nil {
		return httpx.Err(c, fiber.StatusUnprocessableEntity, err.Error())
	}
	return httpx.OK(c, fiber.Map{"best": toRouteDTO(best), "alt": toRouteDTO(alt)})
}

type priceReq struct {
	planner.PlanInput
	Rates map[string]float64 `json:"rates"`
}

// Price handles POST /plan/price: re-compute the cost breakdown for a fixed
// order (called on every tier/mode toggle — pure and fast).
func (h *PlanHandler) Price(c *fiber.Ctx) error {
	var req priceReq
	if err := c.BodyParser(&req); err != nil {
		return httpx.Err(c, fiber.StatusBadRequest, "invalid request body")
	}
	if req.DisplayCurrency == "" {
		return httpx.Err(c, fiber.StatusBadRequest, "display_currency is required")
	}
	b, err := h.svc.Price(req.PlanInput, req.Rates)
	if err != nil {
		return httpx.Err(c, fiber.StatusUnprocessableEntity, err.Error())
	}
	return httpx.OK(c, b)
}

type legsReq struct {
	Legs []planner.LegInput `json:"legs"`
}

// Legs handles POST /plan/legs: per-leg transport mode comparison.
func (h *PlanHandler) Legs(c *fiber.Ctx) error {
	var req legsReq
	if err := c.BodyParser(&req); err != nil {
		return httpx.Err(c, fiber.StatusBadRequest, "invalid request body")
	}
	if len(req.Legs) == 0 {
		return httpx.Err(c, fiber.StatusBadRequest, "legs are required")
	}
	res, err := h.svc.CompareLegs(req.Legs)
	if err != nil {
		return httpx.Err(c, fiber.StatusUnprocessableEntity, err.Error())
	}
	return httpx.OK(c, res)
}
