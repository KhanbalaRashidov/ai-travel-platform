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

	// AI. Keys are read from the environment only — never hardcoded or committed.
	// LLMProvider selects the backend ("gemini" default, free; or "claude").
	// When the selected provider's key is empty, the AI endpoints report "not
	// configured" and the rest of the platform runs normally.
	LLMProvider     string
	GeminiAPIKey    string
	GeminiModel     string
	AnthropicAPIKey string
	AIBudgetUSD     float64
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

		LLMProvider:     getenv("LLM_PROVIDER", "gemini"),
		GeminiAPIKey:    os.Getenv("GEMINI_API_KEY"),
		GeminiModel:     getenv("GEMINI_MODEL", "gemini-2.0-flash"),
		AnthropicAPIKey: os.Getenv("ANTHROPIC_API_KEY"),
		AIBudgetUSD:     getenvFloat("AI_BUDGET_USD", 100),
	}
}

func getenvFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
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
