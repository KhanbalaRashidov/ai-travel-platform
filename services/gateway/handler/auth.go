package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/httpx"
	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/tenant"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/gateway/service"
)

// AuthHandler exposes registration/login/me endpoints.
type AuthHandler struct {
	svc *service.AuthService
}

// NewAuthHandler builds an AuthHandler.
func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

type registerReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResp struct {
	Token string `json:"token"`
	User  any    `json:"user"`
}

// Register handles POST /auth/register.
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req registerReq
	if err := c.BodyParser(&req); err != nil {
		return httpx.Err(c, fiber.StatusBadRequest, "invalid request body")
	}
	user, token, err := h.svc.Register(c.Context(), tenant.FromContext(c), req.Email, req.Password, req.FullName)
	switch {
	case errors.Is(err, service.ErrValidation):
		return httpx.Err(c, fiber.StatusBadRequest, "email required and password must be at least 8 characters")
	case errors.Is(err, service.ErrEmailTaken):
		return httpx.Err(c, fiber.StatusConflict, "email already registered")
	case err != nil:
		return httpx.Err(c, fiber.StatusInternalServerError, "could not register")
	}
	return httpx.Created(c, authResp{Token: token, User: user})
}

// Login handles POST /auth/login.
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req loginReq
	if err := c.BodyParser(&req); err != nil {
		return httpx.Err(c, fiber.StatusBadRequest, "invalid request body")
	}
	user, token, err := h.svc.Login(c.Context(), tenant.FromContext(c), req.Email, req.Password)
	switch {
	case errors.Is(err, service.ErrInvalidCredentials):
		return httpx.Err(c, fiber.StatusUnauthorized, "invalid email or password")
	case err != nil:
		return httpx.Err(c, fiber.StatusInternalServerError, "could not log in")
	}
	return httpx.OK(c, authResp{Token: token, User: user})
}

// Me handles GET /me (requires auth).
func (h *AuthHandler) Me(c *fiber.Ctx) error {
	userID, _ := c.Locals("userID").(string)
	user, err := h.svc.Me(c.Context(), tenant.FromContext(c), userID)
	if err != nil {
		return httpx.Err(c, fiber.StatusNotFound, "user not found")
	}
	return httpx.OK(c, user)
}
