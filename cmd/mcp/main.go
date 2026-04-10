package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/config"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/database"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/jobs"
	internalmcp "github.com/bryanperez/laguna-escondida-marketing/backend/internal/mcp"
	"github.com/bryanperez/laguna-escondida-marketing/backend/migrations"
	"go.uber.org/zap"
	"gorm.io/gorm/logger"
)

func main() {
	// MCP uses stdout for the JSON-RPC protocol — all logs must go to stderr.
	log.SetOutput(os.Stderr)
	log.Println("Starting AI Providers MCP Server...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
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

	zapLogger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer zapLogger.Sync() //nolint:errcheck

	srv, err := internalmcp.New(cfg, jobManager, zapLogger)
	if err != nil {
		log.Fatalf("Failed to create MCP server: %v", err)
	}

	// Graceful shutdown on signal.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		log.Println("MCP server shutting down...")
		os.Exit(0)
	}()

	log.Println("✓ MCP server started (stdio transport)")
	if err := srv.Serve(); err != nil {
		log.Fatalf("MCP server error: %v", err)
	}
}
