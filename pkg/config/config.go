// Package config loads application configuration from environment variables.
package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration. Values come from the environment;
// a local .env file (if present) is loaded for convenience in development.
type Config struct {
	AppEnv      string
	Port        string
	DatabaseURL string
	RedisURL    string
	JWTSecret   string
	JWTTTL      time.Duration
}

// Load reads configuration from the environment, applying sensible dev defaults.
func Load() Config {
	// Best-effort: ignore error when no .env file exists (e.g. in containers).
	_ = godotenv.Load()

	return Config{
		AppEnv:      getenv("APP_ENV", "development"),
		Port:        getenv("API_PORT", "8080"),
		DatabaseURL: getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/ai_travel?sslmode=disable"),
		RedisURL:    getenv("REDIS_URL", "redis://localhost:6379/0"),
		JWTSecret:   getenv("JWT_SECRET", "dev-insecure-change-me"),
		JWTTTL:      time.Duration(getenvInt("JWT_TTL_HOURS", 72)) * time.Hour,
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
