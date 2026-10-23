package main

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/auth"
	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/cache"
	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/tenant"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/ai"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/connectors/estimate"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/connectors/places"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/connectors/poi"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/dataflow"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/gateway/handler"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/gateway/repository"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/gateway/service"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/knowledge"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/planner"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/pricefetch"
)

// buildApp constructs the Fiber app and wires handler -> service -> repository.
// aiPrimary/aiBulk are nil when the Anthropic key is not configured.
func buildApp(pool *pgxpool.Pool, rdb *redis.Client, jwtMgr *auth.JWTManager, defaultTenantID string, aiPrimary, aiBulk ai.LLM, priceSources, corsOrigins string, rateMax int) *fiber.App {
	app := fiber.New(fiber.Config{AppName: "ai-travel-gateway"})
	app.Use(
		recover.New(),
		requestid.New(), // X-Request-Id on every request/response (traceability)
		helmet.New(),    // security headers (X-Content-Type-Options, X-Frame-Options, ...)
		logger.New(logger.Config{Format: "[${time}] ${locals:requestid} ${status} ${method} ${path} ${latency}\n"}),
		cors.New(cors.Config{AllowOrigins: corsOrigins}),
	)
	app.Use(tenant.Middleware(defaultTenantID))

	// Rate limiter for the expensive AI + scrape endpoints (per-IP, per-minute).
	if rateMax <= 0 {
		rateMax = 60
	}
	rl := limiter.New(limiter.Config{Max: rateMax, Expiration: time.Minute})

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

	// Single pricing source of truth: flywheel (transport) + estimator (hotels)
	// + FX, priced server-side. Baseline FX (EUR numeraire); the FX connector
	// refines rates when wired.
	conv := planner.StaticConverter{Rates: map[string]float64{
		"EUR": 1, "USD": 0.92, "AZN": 0.53, "TRY": 0.028, "RUB": 0.010, "GBP": 1.17,
	}}
	placesClient := places.New(nil)
	// Real multi-source fare resolver, driven by the PRICE_SOURCES config. HTTP
	// first; a headless renderer is linked only in the `-tags headless` build
	// (NewHeadless returns nil otherwise). Blocked/empty sources are skipped and
	// the planner falls back to the labeled estimate. The verbatim guardrail
	// ensures the LLM can only report prices present in the fetched page.
	priceSrcs := pricefetch.DefaultSources(priceSources, pricefetch.NewHTTPRenderer(nil), pricefetch.NewHeadless())
	fares := pricefetch.NewResolver(priceSrcs, aiPrimary, conv, kv, 0).
		WithWarm(pricefetch.NewPgWarm(pool))
	dataH := handler.NewDataHandler(routeFW, poiSrc, kbStore, kv, conv, geo, placesClient, fares, aiPrimary)

	// Attach the planner + flywheel as LLM tools to the chat orchestrator.
	aiH := handler.NewAIHandler(aiPrimary, aiBulk,
		NewSequenceTool(estimator),
		NewQuoteTool(routeFW, defaultTenantID),
	)
	analyticsH := handler.NewAnalyticsHandler()

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

	// Real-data flywheel: cached transport quotes, dynamic destinations, POIs,
	// and the full priced plan (single pricing source of truth).
	v1.Get("/data/transport", dataH.Transport)
	v1.Get("/data/destinations", dataH.Destinations)
	v1.Get("/data/poi", dataH.POIs)
	v1.Get("/data/places", rl, dataH.Places)
	v1.Post("/data/plan", rl, dataH.Plan)

	// AI (Claude): itinerary writer + chat + conversational refine loop.
	// 503 when the key is not configured.
	v1.Post("/ai/itinerary", rl, aiH.Itinerary)
	v1.Post("/ai/roadmap", rl, aiH.Roadmap)
	v1.Post("/ai/chat", rl, aiH.Chat)
	v1.Post("/ai/refine", rl, aiH.Refine)

	// Privacy-safe aggregate usage analytics (allowlisted events only).
	v1.Post("/analytics/event", analyticsH.Event)
	v1.Get("/analytics/stats", analyticsH.Stats)

	return app
}
