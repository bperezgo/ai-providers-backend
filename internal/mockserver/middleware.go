package mockserver

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"
)

// statusRecorder wraps http.ResponseWriter to capture the status code.
type statusRecorder struct {
	http.ResponseWriter
	code int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.code = code
	sr.ResponseWriter.WriteHeader(code)
}

// Logger is middleware that logs every request with method, path, status, and duration.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, code: http.StatusOK}

		next.ServeHTTP(rec, r)

		log.Printf("[MOCK] %s %s → %d (%dms)", r.Method, r.URL.Path, rec.code, time.Since(start).Milliseconds())
	})
}

// ErrorInjector is middleware that randomly injects errors based on configured rates.
func ErrorInjector(cfg *Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate latency
		if cfg.LatencyMaxMs > 0 {
			minMs := cfg.LatencyMinMs
			maxMs := cfg.LatencyMaxMs
			if minMs > maxMs {
				minMs = maxMs
			}
			delay := minMs
			if maxMs > minMs {
				delay = minMs + rand.Intn(maxMs-minMs)
			}
			time.Sleep(time.Duration(delay) * time.Millisecond)
		}

		// Error injection
		roll := rand.Float64()

		if roll < cfg.Error429Rate {
			w.Header().Set("Retry-After", "1")
			writeError(w, http.StatusTooManyRequests, "rate limit exceeded (mock)")
			return
		}

		if roll < cfg.Error429Rate+cfg.Error500Rate {
			writeError(w, http.StatusInternalServerError, "internal server error (mock)")
			return
		}

		if roll < cfg.ErrorRate {
			codes := []int{http.StatusBadRequest, http.StatusForbidden, http.StatusServiceUnavailable}
			code := codes[rand.Intn(len(codes))]
			writeError(w, code, fmt.Sprintf("simulated %d error (mock)", code))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]any{
		"error":   msg,
		"status":  code,
		"mock":    true,
	})
}
