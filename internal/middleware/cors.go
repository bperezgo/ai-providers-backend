package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS creates a CORS middleware
func CORS(allowedOrigins string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Check if origin is allowed
		if isOriginAllowed(origin, allowedOrigins) {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// isOriginAllowed checks if an origin is in the allowed list
func isOriginAllowed(origin, allowedOrigins string) bool {
	if origin == "" {
		return false
	}

	// Split allowed origins by comma
	origins := strings.Split(allowedOrigins, ",")

	for _, allowed := range origins {
		allowed = strings.TrimSpace(allowed)

		// Exact match
		if allowed == origin {
			return true
		}

		// Wildcard match (e.g., http://localhost:*)
		if strings.Contains(allowed, "*") {
			prefix := strings.Split(allowed, "*")[0]
			if strings.HasPrefix(origin, prefix) {
				return true
			}
		}
	}

	return false
}
