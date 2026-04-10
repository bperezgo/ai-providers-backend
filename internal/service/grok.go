package service

import (
	"context"
	"fmt"
	"time"

	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/config"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/jobs"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/port"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/providers"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/providers/grok"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/types"
	"github.com/bryanperez/laguna-escondida-marketing/backend/pkg/common"
	pkgservice "github.com/bryanperez/laguna-escondida-marketing/backend/pkg/service"
)

// ---------------------------------------------------------------------------
// Request types
// ---------------------------------------------------------------------------

// GrokTextToVideoRequest is the service-level request for text-to-video generation.
type GrokTextToVideoRequest struct {
	Prompt      string
	SourceImage string // optional source image URL
	Duration    string
	AspectRatio string
	Resolution  string
}

// GrokImageToVideoRequest is the service-level request for image-to-video generation.
type GrokImageToVideoRequest struct {
	ImageURL    string
	Prompt      string
	Duration    string
	AspectRatio string
	Resolution  string
}

// GrokReferenceVideoRequest is the service-level request for reference-image video generation.
type GrokReferenceVideoRequest struct {
	Prompt             string
	ReferenceImageURLs []string
	Duration           string
	AspectRatio        string
	Resolution         string
}

// ---------------------------------------------------------------------------
// Adapters (provider → JobResult)
// ---------------------------------------------------------------------------

type grokT2VAdapter struct {
	provider providers.ImageToVideoProvider // Client implements ImageToVideoProvider
}

func (a *grokT2VAdapter) Process(ctx context.Context, req *GrokTextToVideoRequest) (*port.JobResult, error) {
	result, err := a.provider.GenerateVideo(ctx, req.SourceImage, req.Prompt, providers.ImageToVideoOptions{
		AspectRatio: req.AspectRatio, Duration: req.Duration, Mode: req.Resolution,
	})
	if err != nil {
		return nil, err
	}
	return &port.JobResult{
		ResultURL:  result.LocalPath,
		ResultData: map[string]any{"video_url": result.ProviderURL, "video_path": result.LocalPath},
	}, nil
}

type grokI2VAdapter struct {
	provider providers.ImageToVideoProvider
}

func (a *grokI2VAdapter) Process(ctx context.Context, req *GrokImageToVideoRequest) (*port.JobResult, error) {
	result, err := a.provider.GenerateVideo(ctx, req.ImageURL, req.Prompt, providers.ImageToVideoOptions{
		AspectRatio: req.AspectRatio, Duration: req.Duration, Mode: req.Resolution,
	})
	if err != nil {
		return nil, err
	}
	return &port.JobResult{
		ResultURL:  result.LocalPath,
		ResultData: map[string]any{"video_url": result.ProviderURL, "video_path": result.LocalPath},
	}, nil
}

type grokRefAdapter struct {
	provider providers.ReferenceImageVideoProvider
}

func (a *grokRefAdapter) Process(ctx context.Context, req *GrokReferenceVideoRequest) (*port.JobResult, error) {
	result, err := a.provider.GenerateVideoFromReferences(ctx, req.Prompt, req.ReferenceImageURLs, providers.ReferenceImageVideoOptions{
		AspectRatio: req.AspectRatio, Duration: req.Duration, Mode: req.Resolution,
	})
	if err != nil {
		return nil, err
	}
	return &port.JobResult{
		ResultURL:  result.LocalPath,
		ResultData: map[string]any{"video_url": result.ProviderURL, "video_path": result.LocalPath},
	}, nil
}

// ---------------------------------------------------------------------------
// GrokService exposes all three Grok video generation modes.
// ---------------------------------------------------------------------------

type GrokService struct {
	jobManager *jobs.Manager
	t2v        pkgservice.JobProcessor[GrokTextToVideoRequest]
	i2v        pkgservice.JobProcessor[GrokImageToVideoRequest]
	ref        pkgservice.JobProcessor[GrokReferenceVideoRequest]
}

func NewGrokService(cfg *config.Config, jobManager *jobs.Manager) (*GrokService, error) {
	apiKey, err := cfg.GetAPIKey("grok")
	if err != nil {
		return nil, fmt.Errorf("xAI Grok API key not configured: %w", err)
	}
	httpClient := common.NewHTTPClient(
		time.Duration(cfg.HTTPTimeout)*time.Second,
		cfg.HTTPRetryMax,
		time.Duration(cfg.HTTPRetryWait)*time.Second,
	)
	outDir := cfg.OutputDir + "/grok"

	t2vClient := grok.NewClient(apiKey, httpClient, outDir, cfg.XAIBaseURL)
	i2vClient := grok.NewImageToVideoClient(apiKey, httpClient, outDir, cfg.XAIBaseURL)
	refClient := grok.NewReferenceVideoClient(apiKey, httpClient, outDir, cfg.XAIBaseURL)

	return &GrokService{
		jobManager: jobManager,
		t2v:        pkgservice.NewJobProcessor(jobManager, &grokT2VAdapter{provider: t2vClient}),
		i2v:        pkgservice.NewJobProcessor(jobManager, &grokI2VAdapter{provider: i2vClient}),
		ref:        pkgservice.NewJobProcessor(jobManager, &grokRefAdapter{provider: refClient}),
	}, nil
}

// GenerateTextToVideo creates a job for text-to-video (optionally with a source image).
func (s *GrokService) GenerateTextToVideo(ctx context.Context, req *GrokTextToVideoRequest) (string, error) {
	jobID, err := s.jobManager.CreateJob(ctx, types.ProviderGrok, map[string]any{
		"prompt": req.Prompt, "source_image": req.SourceImage,
		"duration": req.Duration, "aspect_ratio": req.AspectRatio, "resolution": req.Resolution,
	})
	if err != nil {
		return "", err
	}
	go s.t2v.Run(ctx, jobID, req)
	return jobID, nil
}

// GenerateImageToVideo creates a job for image-to-video (image becomes first frame).
func (s *GrokService) GenerateImageToVideo(ctx context.Context, req *GrokImageToVideoRequest) (string, error) {
	jobID, err := s.jobManager.CreateJob(ctx, types.ProviderGrokI2V, map[string]any{
		"image_url": req.ImageURL, "prompt": req.Prompt,
		"duration": req.Duration, "aspect_ratio": req.AspectRatio, "resolution": req.Resolution,
	})
	if err != nil {
		return "", err
	}
	go s.i2v.Run(ctx, jobID, req)
	return jobID, nil
}

// GenerateReferenceVideo creates a job for reference-image video generation.
func (s *GrokService) GenerateReferenceVideo(ctx context.Context, req *GrokReferenceVideoRequest) (string, error) {
	jobID, err := s.jobManager.CreateJob(ctx, types.ProviderGrokRef, map[string]any{
		"prompt": req.Prompt, "reference_image_urls": req.ReferenceImageURLs,
		"duration": req.Duration, "aspect_ratio": req.AspectRatio, "resolution": req.Resolution,
	})
	if err != nil {
		return "", err
	}
	go s.ref.Run(ctx, jobID, req)
	return jobID, nil
}
