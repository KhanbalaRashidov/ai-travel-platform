// Package tenant resolves the active tenant for a request. The platform is
// multi-tenant-ready; for now every request falls back to the default tenant
// unless an authenticated token already set one.
package tenant

import "github.com/gofiber/fiber/v2"

// DefaultSlug is the slug of the single tenant used in the current single-tenant phase.
const DefaultSlug = "default"

// Middleware sets the tenant in request locals. If an upstream (auth) middleware
// already set "tenantID", it is kept; otherwise the default tenant is applied.
func Middleware(defaultTenantID string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if v, ok := c.Locals("tenantID").(string); !ok || v == "" {
			c.Locals("tenantID", defaultTenantID)
		}
		return c.Next()
	}
}

// FromContext returns the resolved tenant ID for the request, or "" if unset.
func FromContext(c *fiber.Ctx) string {
	if v, ok := c.Locals("tenantID").(string); ok {
		return v
	}
	return ""
}
