// Command gateway is the HTTP entry point of the modular monolith. It wires
// configuration, the database pool, Redis, and the feature handlers together,
// then serves the API.
package main

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/auth"
	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/cache"
	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/config"
	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/db"
	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/tenant"
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

	app := buildApp(pool, rdb, jwtMgr, defaultTenantID)

	log.Printf("gateway listening on :%s (env=%s)", cfg.Port, cfg.AppEnv)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("listen: %v", err)
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
