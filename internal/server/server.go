package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"runtime"
	"time"

	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/config"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/handlers"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/jobs"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/middleware"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/service"
	"github.com/bryanperez/laguna-escondida-marketing/backend/pkg/eval"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.uber.org/zap"
)

// Server represents the HTTP server
type Server struct {
	config               *config.Config
	router               *gin.Engine
	httpServer           *http.Server
	jobManager           *jobs.Manager
	logger               *zap.Logger
	healthHandler        *handlers.HealthHandler
	jobsHandler          *handlers.JobsHandler
	soundHandler         *handlers.SoundHandler
	grokHandler          *handlers.GrokHandler
	nanoBananaHandler    *handlers.NanoBananaHandler
	businessHandler      *handlers.BusinessHandler
	courseHandler         *handlers.CourseHandler
	chapterHandler       *handlers.CourseChapterHandler
	planGeneratorHandler *handlers.PlanGeneratorHandler
	aiChatHandler        *handlers.AIChatHandler
}

// New creates a new server instance
func New(cfg *config.Config, jobManager *jobs.Manager, db *gorm.DB, logger *zap.Logger, evalMetrics *eval.EvalMetrics) (*Server, error) {
	gin.SetMode(cfg.GinMode)

	// Create shared services (single source of truth for business logic).
	soundSvc, err := service.NewSoundService(cfg, jobManager)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize sound service: %w", err)
	}

	grokSvc, err := service.NewGrokService(cfg, jobManager)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Grok service: %w", err)
	}

	var nbOpts []eval.DecoratorOption
	if evalMetrics != nil {
		nbOpts = append(nbOpts, eval.WithMetrics(evalMetrics))
	}
	nanoBananaSvc, err := service.NewNanoBananaService(cfg, jobManager, logger, nbOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize NanoBanana service: %w", err)
	}

	// Create platform services (CRUD)
	businessSvc := service.NewBusinessService(db)
	courseSvc := service.NewCourseService(db)
	chapterSvc := service.NewCourseChapterService(db, jobManager)

	// Plan generator service (Claude API for production plan generation)
	// Resolve prompt path relative to the binary's source directory
	_, thisFile, _, _ := runtime.Caller(0)
	backendRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	promptPath := filepath.Join(backendRoot, "prompts", "video-course-plan.md")

	var planGenHandler *handlers.PlanGeneratorHandler
	planGenSvc, err := service.NewPlanGeneratorService(cfg, db, logger, promptPath)
	if err != nil {
		logger.Warn("Plan generator service unavailable (non-fatal)", zap.Error(err))
	} else {
		planGenHandler = handlers.NewPlanGeneratorHandler(planGenSvc)
	}

	// AI chat assistant service (shares Claude API key with plan generator)
	var aiChatHandler *handlers.AIChatHandler
	aiChatSvc, err := service.NewAIChatService(cfg, db, logger)
	if err != nil {
		logger.Warn("AI chat service unavailable (non-fatal)", zap.Error(err))
	} else {
		aiChatHandler = handlers.NewAIChatHandler(aiChatSvc)
	}

	server := &Server{
		config:               cfg,
		router:               gin.New(),
		jobManager:           jobManager,
		logger:               logger,
		healthHandler:        handlers.NewHealthHandler(),
		jobsHandler:          handlers.NewJobsHandler(jobManager),
		soundHandler:         handlers.NewSoundHandler(soundSvc),
		grokHandler:          handlers.NewGrokHandler(grokSvc),
		nanoBananaHandler:    handlers.NewNanoBananaHandler(nanoBananaSvc),
		businessHandler:      handlers.NewBusinessHandler(businessSvc),
		courseHandler:         handlers.NewCourseHandler(courseSvc),
		chapterHandler:       handlers.NewCourseChapterHandler(chapterSvc),
		planGeneratorHandler: planGenHandler,
		aiChatHandler:        aiChatHandler,
	}

	server.setupMiddleware()
	server.setupRoutes()

	return server, nil
}

// setupMiddleware configures all middleware
func (s *Server) setupMiddleware() {
	// Recovery middleware (must be first)
	s.router.Use(middleware.Recovery())

	s.router.Use(otelgin.Middleware("ai-providers-backend"))

	s.router.Use(middleware.Logger(s.logger))

	s.router.Use(middleware.CORS(s.config.CORSAllowOrigins))
}

// setupRoutes configures all routes
func (s *Server) setupRoutes() {
	// Health check endpoint (no prefix)
	s.router.GET("/health", s.healthHandler.Check)

	// Prometheus metrics endpoint
	s.router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		// Health check
		v1.GET("/health", s.healthHandler.Check)

		// Job endpoints
		v1.GET("/jobs", s.jobsHandler.ListJobs)
		v1.GET("/jobs/:job_id/status", s.jobsHandler.GetStatus)
		v1.GET("/jobs/:job_id/result", s.jobsHandler.GetResult)
		v1.GET("/jobs/:job_id/sse", s.jobsHandler.StreamSSE)

		// Sounds endpoints
		v1.POST("/sounds/tts", s.soundHandler.GenerateSpeech)
		v1.POST("/sounds/sound-effects", s.soundHandler.GenerateSoundEffects)
		v1.POST("/sounds/music", s.soundHandler.GenerateMusic)
		v1.GET("/sounds/voices", s.soundHandler.GetVoices)

		// Grok (xAI) endpoints
		v1.POST("/grok/generate", s.grokHandler.GenerateTextToVideo)
		v1.POST("/grok/image-to-video", s.grokHandler.GenerateImageToVideo)
		v1.POST("/grok/reference-video", s.grokHandler.GenerateReferenceVideo)

		// Nano Banana endpoints (via Fal.ai)
		v1.POST("/nanobanana/generate", s.nanoBananaHandler.GenerateImage)

		// Business endpoints
		v1.POST("/businesses", s.businessHandler.Create)
		v1.GET("/businesses", s.businessHandler.List)
		v1.GET("/businesses/:business_id", s.businessHandler.Get)
		v1.PUT("/businesses/:business_id", s.businessHandler.Update)
		v1.DELETE("/businesses/:business_id", s.businessHandler.Delete)

		// Course endpoints
		v1.POST("/businesses/:business_id/courses", s.courseHandler.Create)
		v1.GET("/businesses/:business_id/courses", s.courseHandler.ListByBusiness)
		v1.GET("/courses/:course_id", s.courseHandler.Get)
		v1.PUT("/courses/:course_id", s.courseHandler.Update)
		v1.DELETE("/courses/:course_id", s.courseHandler.Delete)

		// Chapter endpoints
		v1.POST("/courses/:course_id/chapters", s.chapterHandler.Create)
		v1.GET("/courses/:course_id/chapters", s.chapterHandler.ListByCourse)
		v1.GET("/chapters/:chapter_id", s.chapterHandler.Get)
		v1.PUT("/chapters/:chapter_id", s.chapterHandler.Update)
		v1.PUT("/chapters/:chapter_id/plan", s.chapterHandler.SavePlan)
		v1.GET("/chapters/:chapter_id/plan", s.chapterHandler.GetPlan)
		v1.POST("/chapters/:chapter_id/execute", s.chapterHandler.ExecutePlan)
		v1.GET("/chapters/:chapter_id/jobs", s.chapterHandler.GetJobs)

		// Plan generation endpoints (Claude AI)
		if s.planGeneratorHandler != nil {
			v1.POST("/chapters/:chapter_id/generate-plan", s.planGeneratorHandler.GeneratePlan)
			v1.POST("/chapters/:chapter_id/generate-section", s.planGeneratorHandler.GenerateSection)
			v1.POST("/chapters/:chapter_id/refine-segment", s.planGeneratorHandler.RefineSegment)
		}

		// AI chat assistant endpoint
		if s.aiChatHandler != nil {
			v1.POST("/ai/chat", s.aiChatHandler.Chat)
		}
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%s", s.config.Port)

	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	log.Printf("Starting server on %s", addr)
	log.Printf("Health check: http://localhost%s/health", addr)
	log.Printf("API v1: http://localhost%s/api/v1", addr)

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down server...")

	if s.httpServer == nil {
		return nil
	}

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	log.Println("✓ Server shutdown complete")
	return nil
}
