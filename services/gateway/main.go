// Command gateway is the HTTP entry point of the modular monolith. It wires
// configuration, the database pool, Redis, and the feature handlers together,
// then serves the API.
package main

import (
	"context"
	"log"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/auth"
	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/cache"
	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/config"
	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/db"
	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/tenant"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/ai"
	"github.com/KhanbalaRashidov/ai-travel-platform/services/knowledge"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	pool, err := db.NewPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer pool.Close()

	rdb, err := cache.NewRedis(cfg.RedisURL)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}
	defer func() { _ = rdb.Close() }()

	jwtMgr := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTTTL)
	defaultTenantID := loadDefaultTenantID(ctx, pool)

	// Seed the starter destinations once (no-op if the tenant already has some),
	// so the dynamic city list is populated on a fresh database.
	if defaultTenantID != "" {
		if err := knowledge.Seed(ctx, knowledge.NewPgStore(pool), defaultTenantID); err != nil {
			log.Printf("warning: destination seed failed: %v", err)
		}
	}

	aiPrimary, aiBulk := buildLLMs(cfg)

	app := buildApp(pool, rdb, jwtMgr, defaultTenantID, aiPrimary, aiBulk)

	log.Printf("gateway listening on :%s (env=%s)", cfg.Port, cfg.AppEnv)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("listen: %v", err)
	}
}

// buildLLMs constructs the LLM adapters for the selected provider. Both share
// one budget (fail-closed at the cap) and a meter that logs token/cost per call.
// When the selected provider's key is empty it returns nil,nil and the AI
// endpoints report 503. Default provider is Gemini (free).
func buildLLMs(cfg config.Config) (ai.LLM, ai.LLM) {
	budget := ai.NewBudget(cfg.AIBudgetUSD)
	meter := func(model string, u ai.Usage) {
		log.Printf("ai: model=%s in=%d out=%d cost=$%.4f spent=$%.4f/$%.2f",
			model, u.InputTokens, u.OutputTokens, u.CostUSD, budget.SpentUSD, budget.LimitUSD)
	}

	switch strings.ToLower(cfg.LLMProvider) {
	case "claude":
		if cfg.AnthropicAPIKey == "" {
			log.Printf("AI disabled: provider=claude but ANTHROPIC_API_KEY not set (AI endpoints will return 503)")
			return nil, nil
		}
		primary := ai.NewClaude(cfg.AnthropicAPIKey, ai.ModelPrimary, nil).WithBudget(budget).WithMeter(meter)
		bulk := ai.NewClaude(cfg.AnthropicAPIKey, ai.ModelBulk, nil).WithBudget(budget).WithMeter(meter)
		log.Printf("AI enabled: provider=claude primary=%s bulk=%s budget=$%.2f", ai.ModelPrimary.Name, ai.ModelBulk.Name, cfg.AIBudgetUSD)
		return primary, bulk
	default: // gemini (free)
		if cfg.GeminiAPIKey == "" {
			log.Printf("AI disabled: provider=gemini but GEMINI_API_KEY not set (AI endpoints will return 503)")
			return nil, nil
		}
		llm := ai.NewGemini(cfg.GeminiAPIKey, cfg.GeminiModel, nil).
			WithBudget(budget).WithMeter(meter).WithFallback(cfg.GeminiFallbackModel)
		log.Printf("AI enabled: provider=gemini model=%s fallback=%q (free)", llm.Model(), cfg.GeminiFallbackModel)
		return llm, llm // one free model serves both reasoning and bulk
	}
}

// loadDefaultTenantID looks up the single default tenant used in the current
// single-tenant phase. If migrations have not been applied it logs a warning
// and returns "" so the service can still boot.
func loadDefaultTenantID(ctx context.Context, pool *pgxpool.Pool) string {
	var id string
	if err := pool.QueryRow(ctx, `SELECT id FROM tenants WHERE slug = $1`, tenant.DefaultSlug).Scan(&id); err != nil {
		log.Printf("warning: default tenant not found (did you run migrations?): %v", err)
		return ""
	}
	return id
}
