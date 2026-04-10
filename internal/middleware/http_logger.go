package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// HTTPLogger is the net/http equivalent of the Gin Logger middleware.
// It logs every request with structured fields matching the main backend pattern.
//
// Middleware order: HTTPLogger (outermost) → OIDCMiddleware → Handler
//
// The logger places a mutable *RequestInfo in context so that downstream
// middleware (OIDC) can write user_id/session_id. After the response completes
// the logger reads those fields back from the shared struct.
func HTTPLogger(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			logger.Debug("incoming request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("remote_addr", r.RemoteAddr),
				zap.Bool("has_auth_header", r.Header.Get("Authorization") != ""),
			)

			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = uuid.Must(uuid.NewV7()).String()
			}

			// Shared mutable bag for identity fields set by downstream middleware.
			ri := &RequestInfo{}

			ctx := r.Context()
			ctx = NewRequestInfoContext(ctx, ri)
			ctx = context.WithValue(ctx, RequestIDKey, requestID)
			r = r.WithContext(ctx)
			w.Header().Set("X-Request-ID", requestID)

			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(rw, r)

			// Read identity from the shared struct (populated by OIDC middleware).
			fields := []zap.Field{
				zap.String("service", "ai-providers-backend"),
				zap.String("component", "mcp-http"),
				zap.String("user_id", ri.UserID),
				zap.String("session_id", ri.SessionID),
				zap.String("request_id", requestID),
				zap.Int("status", rw.statusCode),
				zap.Duration("latency", time.Since(start)),
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("client_ip", r.RemoteAddr),
			}

			spanCtx := trace.SpanFromContext(r.Context()).SpanContext()
			if spanCtx.HasTraceID() {
				fields = append(fields, zap.String("trace_id", spanCtx.TraceID().String()))
			}
			if spanCtx.HasSpanID() {
				fields = append(fields, zap.String("span_id", spanCtx.SpanID().String()))
			}

			switch {
			case rw.statusCode >= 400:
				logger.Error("request", fields...)
			default:
				logger.Info("request", fields...)
			}
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
