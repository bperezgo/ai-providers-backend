package mockserver

import (
	"fmt"
	"log"
	"net/http"
)

// New creates and configures the mock server with all provider routes.
func New(cfg *Config) http.Handler {
	mux := http.NewServeMux()

	staticBaseURL := fmt.Sprintf("http://localhost:%s", cfg.Port)

	// Register all provider mock routes
	RegisterStaticRoutes(mux)
	RegisterElevenLabsRoutes(mux, staticBaseURL)
	RegisterFalRoutes(mux, cfg, staticBaseURL)
	RegisterOpenAIRoutes(mux, staticBaseURL)
	RegisterLumaRoutes(mux, cfg, staticBaseURL)
	RegisterAnthropicRoutes(mux)

	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "mock server OK")
	})

	// Wrap with error injection + request logging
	return Logger(ErrorInjector(cfg, mux))
}

// Run starts the mock server.
func Run(cfg *Config) error {
	handler := New(cfg)
	addr := ":" + cfg.Port

	log.Printf("Mock server starting on %s", addr)
	log.Printf("  Error rate:     %.1f%%", cfg.ErrorRate*100)
	log.Printf("  429 rate:       %.1f%%", cfg.Error429Rate*100)
	log.Printf("  500 rate:       %.1f%%", cfg.Error500Rate*100)
	log.Printf("  Latency:        %d-%dms", cfg.LatencyMinMs, cfg.LatencyMaxMs)
	log.Printf("  Poll steps:     %d", cfg.PollSteps)

	return http.ListenAndServe(addr, handler)
}
