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
	LLMProvider         string
	GroqAPIKey          string
	GroqModel           string
	GroqFastModel       string // fast model for simple extraction (price parsing)
	GeminiAPIKey        string
	GeminiModel         string
	GeminiFallbackModel string
	AnthropicAPIKey     string
	AIBudgetUSD         float64

	// PriceSources is the PRICE_SOURCES config (newline list of
	// "name | tos-note | url-template"); empty disables real fetching and the
	// platform falls back to the labeled estimate.
	PriceSources string

	// CORSOrigins is a comma-separated allowlist for browsers ("*" in dev).
	// RateLimitPerMin caps requests/min per IP on the AI + scrape endpoints.
	CORSOrigins     string
	RateLimitPerMin int
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

		LLMProvider:         getenv("LLM_PROVIDER", "groq"),
		GroqAPIKey:          os.Getenv("GROQ_API_KEY"),
		GroqModel:           getenv("GROQ_MODEL", "llama-3.3-70b-versatile"),
		GroqFastModel:       getenv("GROQ_FAST_MODEL", "llama-3.1-8b-instant"),
		GeminiAPIKey:        os.Getenv("GEMINI_API_KEY"),
		GeminiModel:         getenv("GEMINI_MODEL", "gemini-2.0-flash"),
		GeminiFallbackModel: os.Getenv("GEMINI_FALLBACK_MODEL"),
		AnthropicAPIKey:     os.Getenv("ANTHROPIC_API_KEY"),
		AIBudgetUSD:         getenvFloat("AI_BUDGET_USD", 100),
		PriceSources:        os.Getenv("PRICE_SOURCES"),
		CORSOrigins:         getenv("CORS_ORIGINS", "*"),
		RateLimitPerMin:     getenvInt("RATE_LIMIT_PER_MIN", 60),
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
