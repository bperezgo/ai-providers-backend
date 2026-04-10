package config

import (
	"errors"
	"fmt"
	"strings"
)

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	var errs []error

	// Validate database configuration.
	// If DATABASE_URL is set (e.g., on Railway), individual DB_* fields are not required.
	if c.DatabaseURL == "" {
		if c.DBHost == "" {
			errs = append(errs, &ValidationError{Field: "DB_HOST", Message: "database host is required"})
		}
		if c.DBPort == "" {
			errs = append(errs, &ValidationError{Field: "DB_PORT", Message: "database port is required"})
		}
		if c.DBUser == "" {
			errs = append(errs, &ValidationError{Field: "DB_USER", Message: "database user is required"})
		}
		if c.DBPassword == "" {
			errs = append(errs, &ValidationError{Field: "DB_PASSWORD", Message: "database password is required"})
		}
		if c.DBName == "" {
			errs = append(errs, &ValidationError{Field: "DB_NAME", Message: "database name is required"})
		}
	}

	// OIDC validation: if enabled, require issuer, audience, and canonical URL
	if c.OIDCEnabled {
		if c.OIDCIssuerURL == "" {
			errs = append(errs, &ValidationError{Field: "OIDC_ISSUER_URL", Message: "required when OIDC is enabled"})
		}
		if c.OIDCAudience == "" {
			errs = append(errs, &ValidationError{Field: "OIDC_AUDIENCE", Message: "required when OIDC is enabled"})
		}
		if c.MCPCanonicalURL == "" {
			errs = append(errs, &ValidationError{Field: "MCP_CANONICAL_URL", Message: "required when OIDC is enabled"})
		}
	}

	// Note: API keys are optional at startup
	// Endpoints will fail if specific API keys are missing when called

	// Combine all errors
	if len(errs) > 0 {
		var errMessages []string
		for _, err := range errs {
			errMessages = append(errMessages, err.Error())
		}
		return errors.New(strings.Join(errMessages, "; "))
	}

	return nil
}

// HasAPIKey checks if a specific API key is configured
func (c *Config) HasAPIKey(provider string) bool {
	switch provider {
	case "elevenlabs":
		return c.ElevenLabsAPIKey != ""
	case "nanobanana", "fal":
		return c.FalAPIKey != ""
	case "anthropic":
		return c.AnthropicAPIKey != ""
	case "grok", "xai":
		return c.XAIAPIKey != ""
	default:
		return false
	}
}

// GetAPIKey retrieves a specific API key
func (c *Config) GetAPIKey(provider string) (string, error) {
	switch provider {
	case "elevenlabs":
		if c.ElevenLabsAPIKey == "" {
			return "", errors.New("ELEVENLABS_API_KEY not configured")
		}
		return c.ElevenLabsAPIKey, nil
	case "nanobanana", "fal":
		if c.FalAPIKey == "" {
			return "", errors.New("FAL_API_KEY not configured")
		}
		return c.FalAPIKey, nil
	case "anthropic":
		if c.AnthropicAPIKey == "" {
			return "", errors.New("ANTHROPIC_API_KEY not configured")
		}
		return c.AnthropicAPIKey, nil
	case "grok", "xai":
		if c.XAIAPIKey == "" {
			return "", errors.New("XAI_API_KEY not configured")
		}
		return c.XAIAPIKey, nil
	default:
		return "", errors.New("unknown provider: " + provider)
	}
}
