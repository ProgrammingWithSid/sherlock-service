package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

type Config struct {
	// Server
	Port           int
	Environment   string
	AllowedOrigins []string

	// Database
	DatabaseURL string

	// Redis
	RedisURL string

	// GitHub App
	GitHubAppID          int64
	GitHubPrivateKeyPath string
	GitHubWebhookSecret  string

	// GitLab
	GitLabToken string

	// Base URL for callbacks and webhooks
	BaseURL string

	// AI Provider
	AIProvider   string
	ClaudeAPIKey string
	OpenAIAPIKey string

	// Storage
	ReposPath      string
	MaxRepoAgeHours int

	// Limits
	MaxFilesPerReview   int
	MaxConcurrentReviews int
	ReviewTimeoutMs      int

	// Cache
	ReviewCacheTTLHours int

	// Features
	EnableIncrementalReviews bool
	EnableCodebaseIndexing   bool

	// Rust Indexer
	RustIndexerURL string // URL for Rust indexer microservice (optional)
}

func Load() *Config {
	// Load .env from backend directory (2 levels up from backend/cmd/server)
	_ = godotenv.Load("../../.env")

	return &Config{
		Port:           getEnvInt("PORT", 3000),
		Environment:   getEnv("NODE_ENV", "development"),
		AllowedOrigins: getEnvSlice("ALLOWED_ORIGINS", []string{"*"}),

		DatabaseURL: getEnv("DATABASE_URL", "postgres://localhost/sherlock?sslmode=disable"),

		RedisURL: getEnv("REDIS_URL", "redis://localhost:6379"),

		GitHubAppID:          getEnvInt64("GITHUB_APP_ID", 0),
		GitHubPrivateKeyPath:  getEnv("GITHUB_PRIVATE_KEY_PATH", ""),
		GitHubWebhookSecret:   getEnv("GITHUB_WEBHOOK_SECRET", ""),

		GitLabToken: getEnv("GITLAB_TOKEN", ""),

		BaseURL: getEnv("BASE_URL", "http://localhost:3000"),

		AIProvider:   getEnv("AI_PROVIDER", "claude"),
		ClaudeAPIKey: getEnv("CLAUDE_API_KEY", ""),
		OpenAIAPIKey: getEnv("OPENAI_API_KEY", ""),

		ReposPath:      getEnv("REPOS_PATH", "/tmp/sherlock-repos"),
		MaxRepoAgeHours: getEnvInt("MAX_REPO_AGE_HOURS", 24),

		MaxFilesPerReview:   getEnvInt("MAX_FILES_PER_REVIEW", 100),
		MaxConcurrentReviews: getEnvInt("MAX_CONCURRENT_REVIEWS", 5),
		ReviewTimeoutMs:      getEnvInt("REVIEW_TIMEOUT_MS", 300000),

		ReviewCacheTTLHours: getEnvInt("REVIEW_CACHE_TTL_HOURS", 24),

		EnableIncrementalReviews: getEnvBool("ENABLE_INCREMENTAL_REVIEWS", true),
		EnableCodebaseIndexing:   getEnvBool("ENABLE_CODEBASE_INDEXING", false),

		RustIndexerURL: getEnv("RUST_INDEXER_URL", ""), // Optional: http://localhost:8081
	}
}

func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return strings.ToLower(value) == "true" || value == "1"
}

// Validate validates the configuration and returns an error if invalid
func (c *Config) Validate() error {
	var errors []string

	if c.DatabaseURL == "" {
		errors = append(errors, "DATABASE_URL is required")
	}

	if c.RedisURL == "" {
		errors = append(errors, "REDIS_URL is required")
	}

	if c.AIProvider != "openai" && c.AIProvider != "claude" {
		errors = append(errors, "AI_PROVIDER must be 'openai' or 'claude'")
	}

	// Only require API keys in production - in development, they're only needed when reviews run
	if c.Environment == "production" {
		if c.AIProvider == "openai" && c.OpenAIAPIKey == "" {
			errors = append(errors, "OPENAI_API_KEY is required when AI_PROVIDER is 'openai'")
		}

		if c.AIProvider == "claude" && c.ClaudeAPIKey == "" {
			errors = append(errors, "CLAUDE_API_KEY is required when AI_PROVIDER is 'claude'")
		}
	} else {
		// In development, log a warning if API keys are missing
		if c.AIProvider == "openai" && c.OpenAIAPIKey == "" {
			log.Warn().Msg("OPENAI_API_KEY not set - reviews will fail when triggered")
		}

		if c.AIProvider == "claude" && c.ClaudeAPIKey == "" {
			log.Warn().Msg("CLAUDE_API_KEY not set - reviews will fail when triggered")
		}
	}

	if c.Port <= 0 || c.Port > 65535 {
		errors = append(errors, "PORT must be between 1 and 65535")
	}

	if c.ReviewCacheTTLHours < 0 {
		errors = append(errors, "REVIEW_CACHE_TTL_HOURS must be non-negative")
	}

	if len(errors) > 0 {
		return fmt.Errorf("config validation failed: %s", strings.Join(errors, ", "))
	}

	return nil
}

func getEnv(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	// Trim whitespace and newlines that might be accidentally included
	return strings.TrimSpace(value)
}

func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		log.Warn().Str("key", key).Str("value", value).Msg("Invalid integer value, using default")
		return defaultValue
	}
	return intValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		log.Warn().Str("key", key).Str("value", value).Msg("Invalid int64 value, using default")
		return defaultValue
	}
	return intValue
}

func getEnvSlice(key string, defaultValue []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return strings.Split(value, ",")
}
