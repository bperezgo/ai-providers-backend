package middleware

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type contextKey string

const (
	UserIDKey    contextKey = "user_id"
	SessionIDKey contextKey = "session_id"
	RequestIDKey contextKey = "request_id"
)

// Logger creates a request logging middleware that enriches logs with
// entity IDs for structured metadata indexing in Loki.
func Logger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Extract entity IDs from request headers/params
		userID := c.GetHeader("X-User-ID")
		sessionID := c.GetHeader("X-Session-ID")

		// Generate a unique request ID if not provided
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.Must(uuid.NewV7()).String()
		}

		// Enrich context so downstream handlers can log with the same IDs
		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, UserIDKey, userID)
		ctx = context.WithValue(ctx, SessionIDKey, sessionID)
		ctx = context.WithValue(ctx, RequestIDKey, requestID)
		c.Request = c.Request.WithContext(ctx)

		// Set request ID in response header for traceability
		c.Header("X-Request-ID", requestID)

		c.Next()

		if raw != "" {
			path = path + "?" + raw
		}

		fields := []zap.Field{
			// Labels (low cardinality — used by Alloy for Loki labels)
			zap.String("service", "ai-providers-backend"),
			zap.String("component", "http"),

			// Structured metadata (high cardinality — indexed in Loki via Alloy)
			zap.String("user_id", userID),
			zap.String("session_id", sessionID),
			zap.String("request_id", requestID),

			// Log line detail
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
			zap.String("client_ip", c.ClientIP()),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("errors", c.Errors.String()),
		}

		// Extract trace_id from OTel span context for log-to-trace correlation
		spanCtx := trace.SpanFromContext(c.Request.Context()).SpanContext()
		if spanCtx.HasTraceID() {
			fields = append(fields, zap.String("trace_id", spanCtx.TraceID().String()))
		}
		if spanCtx.HasSpanID() {
			fields = append(fields, zap.String("span_id", spanCtx.SpanID().String()))
		}

		switch {
		case c.Writer.Status() >= 400:
			logger.Error("request", fields...)
		default:
			logger.Info("request", fields...)
		}
	}
}

// LogWithContext logs a message enriched with entity IDs from the request context.
// Use this in handlers/services for consistent structured metadata.
func LogWithContext(ctx context.Context, logger *zap.Logger, msg string, fields ...zap.Field) {
	base := []zap.Field{
		zap.String("service", "ai-providers-backend"),
		zap.String("user_id", getFromCtx(ctx, UserIDKey)),
		zap.String("session_id", getFromCtx(ctx, SessionIDKey)),
		zap.String("request_id", getFromCtx(ctx, RequestIDKey)),
	}

	spanCtx := trace.SpanFromContext(ctx).SpanContext()
	if spanCtx.HasTraceID() {
		base = append(base, zap.String("trace_id", spanCtx.TraceID().String()))
	}

	base = append(base, fields...)
	logger.Info(msg, base...)
}

func getFromCtx(ctx context.Context, key contextKey) string {
	if v, ok := ctx.Value(key).(string); ok {
		return v
	}
	return ""
}

// RequestInfo is a mutable struct placed in context by HTTPLogger so that
// downstream middleware (e.g. OIDC) can write identity fields that the
// logger reads back after the response completes.
type RequestInfo struct {
	UserID    string
	SessionID string // email for OIDC
}

type requestInfoKeyType struct{}

var requestInfoKey = requestInfoKeyType{}

// NewRequestInfoContext stores a *RequestInfo in the context.
func NewRequestInfoContext(ctx context.Context, ri *RequestInfo) context.Context {
	return context.WithValue(ctx, requestInfoKey, ri)
}

// RequestInfoFromContext retrieves the *RequestInfo from context (nil if absent).
func RequestInfoFromContext(ctx context.Context) *RequestInfo {
	ri, _ := ctx.Value(requestInfoKey).(*RequestInfo)
	return ri
}
