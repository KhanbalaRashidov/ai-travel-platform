package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/auth"
	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/cache"
	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/tenant"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/ai"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/connectors/estimate"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/connectors/poi"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/dataflow"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/gateway/handler"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/gateway/repository"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/gateway/service"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/knowledge"
)

// buildApp constructs the Fiber app and wires handler -> service -> repository.
// aiPrimary/aiBulk are nil when the Anthropic key is not configured.
func buildApp(pool *pgxpool.Pool, rdb *redis.Client, jwtMgr *auth.JWTManager, defaultTenantID string, aiPrimary, aiBulk ai.LLM) *fiber.App {
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

	// Knowledge base (durable/warm) + data flywheel (hot->warm->cold) wiring.
	kbStore := knowledge.NewPgStore(pool)
	recH := handler.NewRecommendHandler(knowledge.NewService(kbStore))

	kv := cache.NewRedisKV(rdb)
	geo := knowledge.SeedGeocoder()
	estimator := estimate.New(geo, 0)
	routeFW := dataflow.NewRouteFlywheel(kv, kbStore, estimator, 0)
	poiSrc := poi.New(geo, nil, 0)
	dataH := handler.NewDataHandler(routeFW, poiSrc, kbStore)

	// Attach the planner + flywheel as LLM tools to the chat orchestrator.
	aiH := handler.NewAIHandler(aiPrimary, aiBulk,
		NewSequenceTool(estimator),
		NewQuoteTool(routeFW, defaultTenantID),
	)

	v1 := app.Group("/api/v1")
	v1.Post("/auth/register", authH.Register)
	v1.Post("/auth/login", authH.Login)
	v1.Get("/me", auth.RequireAuth(jwtMgr), authH.Me)

	// Cost-optimization engine (pure/stateless, public — advisory product).
	v1.Post("/plan/sequence", planH.Sequence)
	v1.Post("/plan/price", planH.Price)
	v1.Post("/plan/legs", planH.Legs)
	v1.Post("/plan/compare", planH.Compare)

	// Knowledge base: popularity-driven recommendations (data flywheel).
	v1.Get("/recommendations", recH.List)
	v1.Post("/recommendations/visit", recH.Visit)

	// Real-data flywheel: cached transport quotes, dynamic destinations, POIs.
	v1.Get("/data/transport", dataH.Transport)
	v1.Get("/data/destinations", dataH.Destinations)
	v1.Get("/data/poi", dataH.POIs)

	// AI (Claude): itinerary writer + chat + conversational refine loop.
	// 503 when the key is not configured.
	v1.Post("/ai/itinerary", aiH.Itinerary)
	v1.Post("/ai/chat", aiH.Chat)
	v1.Post("/ai/refine", aiH.Refine)

	return app
}
