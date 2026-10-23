// Package handler contains the HTTP layer: it parses requests, calls services,
// and writes responses. It never touches the database directly.
package handler

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/httpx"
)

// HealthHandler reports liveness and readiness.
type HealthHandler struct {
	pool *pgxpool.Pool
	rdb  *redis.Client
}

// NewHealthHandler builds a HealthHandler.
func NewHealthHandler(pool *pgxpool.Pool, rdb *redis.Client) *HealthHandler {
	return &HealthHandler{pool: pool, rdb: rdb}
}

// Health is a cheap liveness probe.
func (h *HealthHandler) Health(c *fiber.Ctx) error {
	return httpx.OK(c, fiber.Map{"status": "ok"})
}

// Ready checks dependencies (Postgres + Redis) and reports readiness.
func (h *HealthHandler) Ready(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 2*time.Second)
	defer cancel()

	dbOK, redisOK := true, true
	if h.pool == nil || h.pool.Ping(ctx) != nil {
		dbOK = false
	}
	if h.rdb == nil || h.rdb.Ping(ctx).Err() != nil {
		redisOK = false
	}

	status := fiber.StatusOK
	if !dbOK || !redisOK {
		status = fiber.StatusServiceUnavailable
	}
	return c.Status(status).JSON(httpx.Response{
		Success: dbOK && redisOK,
		Data:    fiber.Map{"postgres": dbOK, "redis": redisOK},
	})
}
