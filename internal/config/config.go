package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	// Server Configuration
	Port     string
	GinMode  string
	LogLevel string

	// Database
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// API Keys
	ElevenLabsAPIKey string
	FalAPIKey        string
	AnthropicAPIKey  string
	XAIAPIKey        string

	// HTTP Settings
	HTTPTimeout   int
	HTTPRetryMax  int
	HTTPRetryWait int

	// Job Settings
	JobExpirationHours int
	JobCleanupInterval int
	JobRetentionDays   int // 0 = keep forever (disable cleanup)

	// Output Settings
	OutputDir string

	// CORS
	CORSAllowOrigins string

	// Eval Settings
	EvalStrategy   string // "always" | "never" | "sample" (default: "never")
	EvalSampleRate uint64 // used when EvalStrategy="sample"; evaluate every N requests (default: 100)

	// Telemetry
	MetricsExporter string // "stdout" | "otlp" | "prometheus" | "" (default: "" = disabled)
	TracesExporter  string // "otlp" | "" (default: "" = disabled)

	// Provider Base URLs (empty = use provider defaults; set for mock server)
	ElevenLabsBaseURL string // override ElevenLabs API base URL
	FalBaseURL        string // override Fal.ai base URL (NanoBanana)
	AnthropicBaseURL  string // override Anthropic API base URL
	XAIBaseURL        string // override xAI API base URL (Grok)

	// MCP Remote (for deploying as a remote MCP server over HTTP)
	MCPAuthToken  string // Bearer token for authenticating remote MCP requests
	MCPListenAddr string // Listen address for remote MCP server (default: ":8080")
	DatabaseURL   string // Single connection string (alternative to individual DB_* fields)

	// OIDC / OAuth 2.1 configuration for MCP remote server
	OIDCEnabled     bool   // Feature flag: when false, fall back to static token auth
	OIDCIssuerURL   string // Auth0 tenant URL, e.g., "https://YOUR_TENANT.us.auth0.com"
	OIDCAudience    string // API identifier from Auth0, e.g., "https://mcp.laguna-escondida.com"
	MCPCanonicalURL string // Public URL of this MCP server (used in resource metadata)
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if it doesn't)
	_ = godotenv.Load()

	cfg := &Config{
		// Server Configuration
		Port:     getEnv("PORT", "8080"),
		GinMode:  getEnv("GIN_MODE", "release"),
		LogLevel: getEnv("LOG_LEVEL", "info"),

		// Database
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "ai_backend"),
		DBPassword: getEnv("DB_PASSWORD", "ai_backend_password"),
		DBName:     getEnv("DB_NAME", "ai_backend"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),

		// API Keys
		ElevenLabsAPIKey: getEnv("ELEVENLABS_API_KEY", ""),
		FalAPIKey:        getEnv("FAL_API_KEY", ""),
		AnthropicAPIKey:  getEnv("ANTHROPIC_API_KEY", ""),
		XAIAPIKey:        getEnv("XAI_API_KEY", ""),

		// HTTP Settings
		HTTPTimeout:   getEnvAsInt("HTTP_TIMEOUT", 300),
		HTTPRetryMax:  getEnvAsInt("HTTP_RETRY_MAX", 3),
		HTTPRetryWait: getEnvAsInt("HTTP_RETRY_WAIT", 2),

		// Job Settings
		JobExpirationHours: getEnvAsInt("JOB_EXPIRATION_HOURS", 24),
		JobCleanupInterval: getEnvAsInt("JOB_CLEANUP_INTERVAL", 60),
		JobRetentionDays:   getEnvAsInt("JOB_RETENTION_DAYS", 0),

		// Output Settings
		OutputDir: getEnv("OUTPUT_DIR", "./outputs"),

		// CORS
		CORSAllowOrigins: getEnv("CORS_ALLOW_ORIGINS", "http://localhost:*"),

		// Eval Settings
		EvalStrategy:   getEnv("EVAL_STRATEGY", "never"),
		EvalSampleRate: uint64(getEnvAsInt("EVAL_SAMPLE_RATE", 100)),

		// Telemetry
		MetricsExporter: getEnv("OTEL_METRICS_EXPORTER", ""),
		TracesExporter:  getEnv("OTEL_TRACES_EXPORTER", ""),

		// Provider Base URLs
		ElevenLabsBaseURL: getEnv("ELEVENLABS_BASE_URL", ""),
		FalBaseURL:        getEnv("FAL_BASE_URL", ""),
		AnthropicBaseURL:  getEnv("ANTHROPIC_BASE_URL", ""),
		XAIBaseURL:        getEnv("XAI_BASE_URL", ""),

		// MCP Remote
		MCPAuthToken:  getEnv("MCP_AUTH_TOKEN", ""),
		MCPListenAddr: getEnv("MCP_LISTEN_ADDR", ":8080"),
		DatabaseURL:   getEnv("DATABASE_URL", ""),

		// OIDC / OAuth 2.1
		OIDCEnabled:     getEnv("OIDC_ENABLED", "false") == "true",
		OIDCIssuerURL:   getEnv("OIDC_ISSUER_URL", ""),
		OIDCAudience:    getEnv("OIDC_AUDIENCE", ""),
		MCPCanonicalURL: getEnv("MCP_CANONICAL_URL", ""),
	}

	return cfg, nil
}

// GetDSN returns PostgreSQL connection string.
// If DATABASE_URL is set (e.g., on Railway), it is used directly.
func (c *Config) GetDSN() string {
	if c.DatabaseURL != "" {
		return c.DatabaseURL
	}
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode,
	)
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvAsInt gets an environment variable as integer or returns a default value
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}
