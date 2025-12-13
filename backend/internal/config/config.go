package config

import (
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
	GitHubClientID       string
	GitHubClientSecret   string

	// GitLab
	GitLabToken       string
	GitLabClientID    string
	GitLabClientSecret string

	// OAuth
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
}

func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		Port:           getEnvInt("PORT", 3000),
		Environment:   getEnv("NODE_ENV", "development"),
		AllowedOrigins: getEnvSlice("ALLOWED_ORIGINS", []string{"*"}),

		DatabaseURL: getEnv("DATABASE_URL", "postgres://localhost/sherlock?sslmode=disable"),

		RedisURL: getEnv("REDIS_URL", "redis://localhost:6379"),

		GitHubAppID:          getEnvInt64("GITHUB_APP_ID", 0),
		GitHubPrivateKeyPath:  getEnv("GITHUB_PRIVATE_KEY_PATH", ""),
		GitHubWebhookSecret:   getEnv("GITHUB_WEBHOOK_SECRET", ""),
		GitHubClientID:        getEnv("GITHUB_CLIENT_ID", ""),
		GitHubClientSecret:    getEnv("GITHUB_CLIENT_SECRET", ""),

		GitLabToken:       getEnv("GITLAB_TOKEN", ""),
		GitLabClientID:    getEnv("GITLAB_CLIENT_ID", ""),
		GitLabClientSecret: getEnv("GITLAB_CLIENT_SECRET", ""),

		BaseURL: getEnv("BASE_URL", "http://localhost:3000"),

		AIProvider:   getEnv("AI_PROVIDER", "claude"),
		ClaudeAPIKey: getEnv("CLAUDE_API_KEY", ""),
		OpenAIAPIKey: getEnv("OPENAI_API_KEY", ""),

		ReposPath:      getEnv("REPOS_PATH", "/tmp/sherlock-repos"),
		MaxRepoAgeHours: getEnvInt("MAX_REPO_AGE_HOURS", 24),

		MaxFilesPerReview:   getEnvInt("MAX_FILES_PER_REVIEW", 100),
		MaxConcurrentReviews: getEnvInt("MAX_CONCURRENT_REVIEWS", 5),
		ReviewTimeoutMs:      getEnvInt("REVIEW_TIMEOUT_MS", 300000),
	}
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
