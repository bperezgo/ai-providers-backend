// Package mcp provides a Model Context Protocol server that exposes all AI
// provider functionality as MCP tools — without going through HTTP.
package mcp

import (
	"fmt"

	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/config"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/jobs"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/mcp/tools"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/service"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"
)

// Server wraps the MCP server with all AI provider tools registered.
type Server struct {
	srv *mcpserver.MCPServer
}

// New creates and configures the MCP server with all registered tools.
// Services are shared with the HTTP server — same config, job manager, and providers.
func New(cfg *config.Config, jobManager *jobs.Manager, logger *zap.Logger) (*Server, error) {
	s := mcpserver.NewMCPServer("ai-providers", "1.0.0",
		mcpserver.WithToolCapabilities(true),
	)

	// Job management tools — always registered (no API key needed).
	tools.RegisterJobTools(s, jobManager)

	// ElevenLabs: TTS, sound effects, music, voices.
	soundSvc, err := service.NewSoundService(cfg, jobManager)
	if err != nil {
		return nil, fmt.Errorf("sound service: %w", err)
	}
	tools.RegisterSoundTools(s, soundSvc)

	// Grok (xAI) image-to-video.
	grokSvc, err := service.NewGrokService(cfg, jobManager)
	if err != nil {
		return nil, fmt.Errorf("grok service: %w", err)
	}
	tools.RegisterGrokTools(s, grokSvc)

	// Nano Banana text-to-image (with eval decorator, no OTel metrics).
	nanoBananaSvc, err := service.NewNanoBananaService(cfg, jobManager, logger)
	if err != nil {
		return nil, fmt.Errorf("nanobanana service: %w", err)
	}
	tools.RegisterNanoBananaTools(s, nanoBananaSvc)

	return &Server{srv: s}, nil
}

// MCPServer returns the underlying MCP server for use with different transports.
func (s *Server) MCPServer() *mcpserver.MCPServer {
	return s.srv
}

// Serve starts the MCP server on stdio transport (blocking).
func (s *Server) Serve() error {
	return mcpserver.ServeStdio(s.srv)
}
