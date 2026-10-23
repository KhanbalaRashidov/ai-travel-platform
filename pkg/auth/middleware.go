package auth

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// RequireAuth is Fiber middleware that validates the Bearer token and stores
// the resolved userID/tenantID/email in the request locals. Routes mounted
// behind it can rely on those values being present.
func RequireAuth(m *JWTManager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		header := c.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).
				JSON(fiber.Map{"success": false, "error": "missing bearer token"})
		}
		claims, err := m.Verify(strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).
				JSON(fiber.Map{"success": false, "error": "invalid or expired token"})
		}
		c.Locals("userID", claims.UserID)
		c.Locals("tenantID", claims.TenantID)
		c.Locals("email", claims.Email)
		return c.Next()
	}
}
