package common

import (
	"errors"
	"fmt"
)

// Common errors
var (
	ErrAPIKeyMissing      = errors.New("API key missing")
	ErrInvalidRequest     = errors.New("invalid request")
	ErrRateLimitExceeded  = errors.New("rate limit exceeded")
	ErrMaxRetriesExceeded = errors.New("max retries exceeded")
	ErrJobNotFound        = errors.New("job not found")
	ErrJobFailed          = errors.New("job generation failed")
	ErrTimeout            = errors.New("request timeout")
)

// APIError represents an error from an external API
type APIError struct {
	StatusCode int
	Message    string
	Provider   string
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s API error (%d): %s", e.Provider, e.StatusCode, e.Message)
}

// NewAPIError creates a new API error
func NewAPIError(provider string, statusCode int, message string, body string) *APIError {
	return &APIError{
		Provider:   provider,
		StatusCode: statusCode,
		Message:    message,
		Body:       body,
	}
}

// IsRateLimitError checks if the error is a rate limit error
func IsRateLimitError(err error) bool {
	if err == nil {
		return false
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == 429
	}

	return errors.Is(err, ErrRateLimitExceeded)
}

// IsNotFoundError checks if the error is a not found error
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == 404
	}

	return errors.Is(err, ErrJobNotFound)
}
