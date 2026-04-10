package service

import (
	"context"
	"fmt"
	"time"

	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/config"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/jobs"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/port"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/providers"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/providers/nanobanana"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/types"
	"github.com/bryanperez/laguna-escondida-marketing/backend/pkg/common"
	"github.com/bryanperez/laguna-escondida-marketing/backend/pkg/eval"
	"github.com/bryanperez/laguna-escondida-marketing/backend/pkg/judge"
	pkgservice "github.com/bryanperez/laguna-escondida-marketing/backend/pkg/service"
	"go.uber.org/zap"
)

// NanoBananaRequest is the service-level request for NanoBanana image generation.
type NanoBananaRequest struct {
	Prompt            string
	NegativePrompt    string
	ImageSize         string
	NumImages         int
	NumInferenceSteps int
	GuidanceScale     float64
	Seed              int64
}

type nanoBananaAdapter struct {
	provider providers.TextToImageProvider
}

func (a *nanoBananaAdapter) Process(ctx context.Context, req *NanoBananaRequest) (*port.JobResult, error) {
	result, err := a.provider.GenerateImage(ctx, req.Prompt, providers.TextToImageOptions{
		Size: req.ImageSize,
		N:    req.NumImages,
	})
	if err != nil {
		return nil, err
	}
	return &port.JobResult{
		ResultURL: result.LocalPath,
		ResultData: map[string]any{
			"image_url": result.ProviderURL, "image_path": result.LocalPath,
			"width": result.Width, "height": result.Height,
		},
	}, nil
}

// NanoBananaService encapsulates NanoBanana image generation with eval decorator.
type NanoBananaService struct {
	jobManager *jobs.Manager
	processor  pkgservice.JobProcessor[NanoBananaRequest]
}

func NewNanoBananaService(cfg *config.Config, jobManager *jobs.Manager, logger *zap.Logger, evalOpts ...eval.DecoratorOption) (*NanoBananaService, error) {
	apiKey, err := cfg.GetAPIKey("nanobanana")
	if err != nil {
		return nil, fmt.Errorf("Nano Banana (Fal.ai) API key not configured: %w", err)
	}
	httpClient := common.NewHTTPClient(
		time.Duration(cfg.HTTPTimeout)*time.Second,
		cfg.HTTPRetryMax,
		time.Duration(cfg.HTTPRetryWait)*time.Second,
	)
	client := nanobanana.NewClient(apiKey, httpClient, cfg.OutputDir+"/nanobanana", cfg.FalBaseURL)
	judgeClient := judge.NewJudge(cfg.AnthropicAPIKey, common.NewHTTPClient(30*time.Second, 2, time.Second), cfg.AnthropicBaseURL)

	var strategy eval.Strategy
	switch cfg.EvalStrategy {
	case "always":
		strategy = eval.AlwaysEvaluate{}
	case "sample":
		n := cfg.EvalSampleRate
		if n == 0 {
			n = 100
		}
		strategy = eval.NewSampleEvaluate(n)
	default:
		strategy = eval.NeverEvaluate{}
	}

	provider := eval.NewEvalDecorator(client, judgeClient, strategy, "marketing", logger, jobManager, evalOpts...)
	processor := pkgservice.NewJobProcessor(jobManager, &nanoBananaAdapter{provider: provider})
	return &NanoBananaService{jobManager: jobManager, processor: processor}, nil
}

// GenerateImage creates a job and starts async NanoBanana generation. Returns the job ID.
func (s *NanoBananaService) GenerateImage(ctx context.Context, req *NanoBananaRequest) (string, error) {
	jobID, err := s.jobManager.CreateJob(ctx, types.ProviderNanaBanana, map[string]any{
		"prompt": req.Prompt, "image_size": req.ImageSize, "num_images": req.NumImages,
	})
	if err != nil {
		return "", err
	}
	go s.processor.Run(ctx, jobID, req)
	return jobID, nil
}
