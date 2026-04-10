package common

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// HTTPClient is a wrapper around http.Client with retry logic
type HTTPClient struct {
	client     *http.Client
	maxRetries int
	retryWait  time.Duration
}

// NewHTTPClient creates a new HTTP client with retry capabilities
func NewHTTPClient(timeout time.Duration, maxRetries int, retryWait time.Duration) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: timeout,
		},
		maxRetries: maxRetries,
		retryWait:  retryWait,
	}
}

// Do executes an HTTP request with retry logic
func (c *HTTPClient) Do(req *http.Request) (*http.Response, error) {
	return c.DoWithRetry(req.Context(), req)
}

// DoWithRetry executes an HTTP request with retry logic and context
func (c *HTTPClient) DoWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		// Clone request for retry (body needs to be re-readable)
		reqClone := req.Clone(ctx)

		// Execute request
		resp, err = c.client.Do(reqClone)

		// Success - return immediately
		if err == nil && resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}

		// Rate limit (429) - parse Retry-After header
		if resp != nil && resp.StatusCode == http.StatusTooManyRequests {
			retryAfter := c.parseRetryAfter(resp)
			if attempt < c.maxRetries {
				select {
				case <-time.After(retryAfter):
					continue
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
		}

		// Server error (5xx) or network error - retry with exponential backoff
		if attempt < c.maxRetries {
			backoff := c.retryWait * time.Duration(attempt+1)
			select {
			case <-time.After(backoff):
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	// Max retries exceeded
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMaxRetriesExceeded, err)
	}

	return resp, nil
}

// parseRetryAfter parses the Retry-After header from a 429 response
func (c *HTTPClient) parseRetryAfter(resp *http.Response) time.Duration {
	retryAfterHeader := resp.Header.Get("Retry-After")
	if retryAfterHeader == "" {
		return 5 * time.Second // Default retry after 5 seconds
	}

	// Try parsing as seconds
	if seconds, err := strconv.Atoi(retryAfterHeader); err == nil {
		return time.Duration(seconds) * time.Second
	}

	// Try parsing as HTTP date
	if retryTime, err := http.ParseTime(retryAfterHeader); err == nil {
		duration := time.Until(retryTime)
		if duration > 0 {
			return duration
		}
	}

	// Default fallback
	return 5 * time.Second
}

// Get executes a GET request with retry logic
func (c *HTTPClient) Get(ctx context.Context, url string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request: %w", err)
	}

	// Add headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return c.DoWithRetry(ctx, req)
}

// Post executes a POST request with retry logic
func (c *HTTPClient) Post(ctx context.Context, url string, body interface{}, headers map[string]string) (*http.Response, error) {
	// This is a simplified version - in reality, you'd need to handle body serialization
	// For now, we'll implement this in the provider-specific clients
	return nil, fmt.Errorf("Post not implemented yet - use DoWithRetry with custom request")
}
