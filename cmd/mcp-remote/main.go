package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/config"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/database"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/jobs"
	internalmcp "github.com/bryanperez/laguna-escondida-marketing/backend/internal/mcp"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/middleware"
	"github.com/bryanperez/laguna-escondida-marketing/backend/migrations"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"
	"gorm.io/gorm/logger"
)

func main() {
	log.Println("Starting AI Providers MCP Remote Server...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	if !cfg.OIDCEnabled && cfg.MCPAuthToken == "" {
		log.Fatalf("MCP_AUTH_TOKEN is required when OIDC is disabled")
	}

	db, err := database.Connect(cfg.GetDSN(), logger.Error)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	database.MigrationFS = migrations.SQLFiles
	database.MigrationDir = migrations.Dir
	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	jobRepo := jobs.NewJobRepository(db)
	broadcaster := jobs.NewSSEBroadcaster()
	jobManager := jobs.NewManager(jobRepo, broadcaster, cfg.JobExpirationHours)
	defer jobManager.Close()

	var zapLogger *zap.Logger
	if os.Getenv("DEBUG") == "true" {
		zapLogger, err = zap.NewDevelopment()
	} else {
		zapLogger, err = zap.NewProduction()
	}
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer zapLogger.Sync() //nolint:errcheck

	srv, err := internalmcp.New(cfg, jobManager, zapLogger)
	if err != nil {
		log.Fatalf("Failed to create MCP server: %v", err)
	}

	// Create Streamable HTTP transport wrapping the MCP server.
	mcpHTTP := mcpserver.NewStreamableHTTPServer(srv.MCPServer(),
		mcpserver.WithStateLess(true),
		mcpserver.WithHeartbeatInterval(30*time.Second),
	)

	// Determine auth strategy.
	var authMW func(http.Handler) http.Handler
	if cfg.OIDCEnabled {
		oidcMW, err := middleware.NewOIDCMiddleware(
			context.Background(),
			cfg.OIDCIssuerURL,
			cfg.OIDCAudience,
			cfg.MCPCanonicalURL+"/.well-known/oauth-protected-resource",
			zapLogger,
		)
		if err != nil {
			log.Fatalf("Failed to initialize OIDC: %v", err)
		}
		authMW = oidcMW.Authenticate
	} else {
		authMW = func(next http.Handler) http.Handler {
			return authMiddleware(cfg.MCPAuthToken, next)
		}
	}

	// Request logging middleware (wraps everything).
	logMW := middleware.HTTPLogger(zapLogger)

	// Build HTTP mux.
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)

	if cfg.OIDCEnabled {
		mux.HandleFunc("/.well-known/oauth-protected-resource", protectedResourceHandler(cfg))
		mux.HandleFunc("/.well-known/oauth-authorization-server", authServerMetadataHandler(cfg))
	}

	// Stack: logging → auth → handler
	mux.Handle("/mcp", logMW(authMW(mcpHTTP)))
	mux.Handle("/files/", logMW(authMW(newFileHandler(jobManager))))

	httpSrv := &http.Server{
		Addr:              cfg.MCPListenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Graceful shutdown.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		log.Println("Shutting down remote MCP server...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpSrv.Shutdown(ctx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
	}()

	authMode := "static-token"
	if cfg.OIDCEnabled {
		authMode = "OIDC (issuer: " + cfg.OIDCIssuerURL + ")"
	}
	log.Printf("✓ Remote MCP server started on %s (Streamable HTTP transport)", cfg.MCPListenAddr)
	log.Printf("  Auth:    %s", authMode)
	log.Printf("  /health  — health check (no auth)")
	log.Printf("  /mcp     — MCP endpoint (auth required)")
	log.Printf("  /files/  — file download (auth required)")
	if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("HTTP server error: %v", err)
	}
}

// authMiddleware validates static Bearer token authentication (fallback when OIDC is disabled).
func authMiddleware(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "Bearer "+token {
			next.ServeHTTP(w, r)
			return
		}
		if q := r.URL.Query().Get("token"); q == token {
			next.ServeHTTP(w, r)
			return
		}
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
}

// authServerMetadataHandler returns OAuth Authorization Server Metadata (RFC 8414).
// Claude Desktop fetches this from the MCP server to discover the real authorization
// server endpoints (Auth0). All URLs point to Auth0 — the MCP server is only a relay
// for metadata discovery.
func authServerMetadataHandler(cfg *config.Config) http.HandlerFunc {
	issuer := strings.TrimRight(cfg.OIDCIssuerURL, "/")
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                issuer + "/",
			"authorization_endpoint":                issuer + "/authorize",
			"token_endpoint":                        issuer + "/oauth/token",
			"registration_endpoint":                 issuer + "/oidc/register",
			"response_types_supported":              []string{"code"},
			"code_challenge_methods_supported":       []string{"S256"},
			"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
			"token_endpoint_auth_methods_supported": []string{"client_secret_post", "client_secret_basic"},
			"scopes_supported":                      []string{"openid", "email", "profile", "offline_access"},
		})
	}
}

// protectedResourceHandler returns the OAuth Protected Resource Metadata (RFC 9728).
// Claude Code fetches this after receiving a 401 to discover the authorization server.
func protectedResourceHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		json.NewEncoder(w).Encode(map[string]any{
			"resource":                 cfg.MCPCanonicalURL,
			"authorization_servers":    []string{cfg.OIDCIssuerURL},
			"scopes_supported":         []string{"openid", "email"},
			"bearer_methods_supported": []string{"header"},
		})
	}
}

// healthHandler returns 200 OK for health checks (no auth required).
func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, `{"status":"ok"}`)
}

// fileHandler serves job result files/URLs.
type fileHandler struct {
	jobManager *jobs.Manager
}

func newFileHandler(jm *jobs.Manager) *fileHandler {
	return &fileHandler{jobManager: jm}
}

func (h *fileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract job ID from /files/{job_id}
	jobID := strings.TrimPrefix(r.URL.Path, "/files/")
	jobID = strings.TrimSuffix(jobID, "/")
	if jobID == "" {
		http.Error(w, "job ID required", http.StatusBadRequest)
		return
	}

	job, err := h.jobManager.GetJob(r.Context(), jobID)
	if err != nil {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}

	if job.Status != "completed" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  job.Status,
			"message": "job not yet completed",
		})
		return
	}

	// If ResultURL is an external URL, redirect to it.
	if job.ResultURL != "" && strings.HasPrefix(job.ResultURL, "http") {
		http.Redirect(w, r, job.ResultURL, http.StatusFound)
		return
	}

	// If ResultURL is a local file path, serve the file.
	if job.ResultURL != "" {
		http.ServeFile(w, r, job.ResultURL)
		return
	}

	// Fall back to returning ResultData as JSON.
	if job.ResultData != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(job.ResultData)
		return
	}

	http.Error(w, "no result available", http.StatusNotFound)
}
