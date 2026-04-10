package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/config"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/database"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/jobs"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/server"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/telemetry"
	"github.com/bryanperez/laguna-escondida-marketing/backend/migrations"
	"github.com/bryanperez/laguna-escondida-marketing/backend/pkg/eval"
	otelruntime "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.uber.org/zap"
	"gorm.io/gorm/logger"
)

func main() {
	log.Println("Starting AI Providers Backend...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	log.Println("✓ Configuration loaded")

	var gormLogLevel logger.LogLevel
	switch cfg.LogLevel {
	case "debug":
		gormLogLevel = logger.Info
	case "info":
		gormLogLevel = logger.Warn
	default:
		gormLogLevel = logger.Error
	}

	db, err := database.Connect(cfg.GetDSN(), gormLogLevel)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Wire embedded SQL migrations and seeds
	database.MigrationFS = migrations.SQLFiles
	database.MigrationDir = migrations.Dir
	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	log.Println("✓ Database ready")

	// Initialize job system
	jobRepo := jobs.NewJobRepository(db)
	broadcaster := jobs.NewSSEBroadcaster()
	jobManager := jobs.NewManager(jobRepo, broadcaster, cfg.JobExpirationHours)
	defer jobManager.Close()

	log.Println("✓ Job manager initialized")

	// Start cleanup worker (only if retention is enabled)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if cfg.JobRetentionDays > 0 {
		cleanupWorker := jobs.NewCleanupWorker(jobManager, cfg.JobCleanupInterval)
		go cleanupWorker.Start(ctx)
		log.Printf("✓ Cleanup worker started (retention: %d days)", cfg.JobRetentionDays)
	} else {
		log.Println("✓ Job cleanup disabled (JOB_RETENTION_DAYS=0)")
	}

	zapLogger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer zapLogger.Sync() //nolint:errcheck

	// Setup OpenTelemetry metrics
	mp, err := telemetry.NewMeterProvider(ctx, cfg.MetricsExporter)
	if err != nil {
		log.Fatalf("Failed to initialize meter provider: %v", err)
	}
	defer func() { _ = mp.Shutdown(context.Background()) }()

	var evalMetrics *eval.EvalMetrics
	if cfg.MetricsExporter != "" {
		meter := mp.Meter("ai-backend/eval")
		evalMetrics, err = eval.NewEvalMetrics(meter)
		if err != nil {
			log.Fatalf("Failed to create eval metrics: %v", err)
		}
		log.Printf("✓ OTel metrics enabled (exporter: %s)", cfg.MetricsExporter)

		// Start runtime metrics (goroutines, GC, memory)
		if err := otelruntime.Start(otelruntime.WithMinimumReadMemStatsInterval(15 * time.Second)); err != nil {
			log.Printf("Warning: failed to start runtime metrics: %v", err)
		}
	} else {
		log.Println("✓ OTel metrics disabled (OTEL_METRICS_EXPORTER not set)")
	}

	// Setup OpenTelemetry tracing
	tp, err := telemetry.NewTracerProvider(ctx, cfg.TracesExporter)
	if err != nil {
		log.Fatalf("Failed to initialize tracer provider: %v", err)
	}
	defer func() { _ = tp.Shutdown(context.Background()) }()

	if cfg.TracesExporter != "" {
		log.Printf("✓ OTel tracing enabled (exporter: %s)", cfg.TracesExporter)
	} else {
		log.Println("✓ OTel tracing disabled (OTEL_TRACES_EXPORTER not set)")
	}

	// Create and start server
	srv, err := server.New(cfg, jobManager, db, zapLogger, evalMetrics)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Handle graceful shutdown
	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	log.Println("✓ Server started successfully")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("\nReceived shutdown signal...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())
	defer shutdownCancel()

	// Stop cleanup worker
	cancel()

	// Shutdown server
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}

	log.Println("✓ Shutdown complete")
}
