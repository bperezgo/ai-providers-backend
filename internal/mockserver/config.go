package mockserver

import (
	"os"
	"strconv"
)

// Config holds mock server configuration.
type Config struct {
	Port         string  // server port (default: "9999")
	ErrorRate    float64 // generic error rate 0.0-1.0 (default: 0)
	Error429Rate float64 // rate-limit error rate 0.0-1.0 (default: 0)
	Error500Rate float64 // internal server error rate 0.0-1.0 (default: 0)
	LatencyMinMs int     // min simulated latency in ms (default: 10)
	LatencyMaxMs int     // max simulated latency in ms (default: 100)
	PollSteps    int     // poll cycles before COMPLETED for async providers (default: 2)
}

// LoadConfig reads mock server config from environment variables.
func LoadConfig() *Config {
	return &Config{
		Port:         getEnv("MOCK_PORT", "9999"),
		ErrorRate:    getEnvAsFloat("MOCK_ERROR_RATE", 0),
		Error429Rate: getEnvAsFloat("MOCK_ERROR_429_RATE", 0),
		Error500Rate: getEnvAsFloat("MOCK_ERROR_500_RATE", 0),
		LatencyMinMs: getEnvAsInt("MOCK_LATENCY_MIN_MS", 10),
		LatencyMaxMs: getEnvAsInt("MOCK_LATENCY_MAX_MS", 100),
		PollSteps:    getEnvAsInt("MOCK_POLL_STEPS", 2),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvAsFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}
