package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/auth"
	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/tenant"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/gateway/handler"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/gateway/repository"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/gateway/service"
)

// buildApp constructs the Fiber app and wires handler -> service -> repository.
func buildApp(pool *pgxpool.Pool, rdb *redis.Client, jwtMgr *auth.JWTManager, defaultTenantID string) *fiber.App {
	app := fiber.New(fiber.Config{AppName: "ai-travel-gateway"})
	app.Use(recover.New(), logger.New(), cors.New())
	app.Use(tenant.Middleware(defaultTenantID))

	health := handler.NewHealthHandler(pool, rdb)
	app.Get("/health", health.Health)
	app.Get("/ready", health.Ready)

	userRepo := repository.NewUserRepository(pool)
	authSvc := service.NewAuthService(userRepo, jwtMgr)
	authH := handler.NewAuthHandler(authSvc)

	planH := handler.NewPlanHandler(service.NewPlanService())

	v1 := app.Group("/api/v1")
	v1.Post("/auth/register", authH.Register)
	v1.Post("/auth/login", authH.Login)
	v1.Get("/me", auth.RequireAuth(jwtMgr), authH.Me)

	// Cost-optimization engine (pure/stateless, public — advisory product).
	v1.Post("/plan/sequence", planH.Sequence)
	v1.Post("/plan/price", planH.Price)

	return app
}
