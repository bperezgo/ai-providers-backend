package service

import (
	"context"
	"fmt"
	"time"

	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/config"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/jobs"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/port"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/providers/elevenlabs"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/types"
	"github.com/bryanperez/laguna-escondida-marketing/backend/pkg/common"
	pkgservice "github.com/bryanperez/laguna-escondida-marketing/backend/pkg/service"
)

type ttsAdapter struct{ client *elevenlabs.Client }

func (a *ttsAdapter) Process(ctx context.Context, req *elevenlabs.TTSRequest) (*port.JobResult, error) {
	result, err := a.client.GenerateSpeech(ctx, req)
	if err != nil {
		return nil, err
	}
	return &port.JobResult{
		ResultURL:  result.AudioURL,
		ResultData: map[string]any{"audio_url": result.AudioURL, "duration": result.Duration},
	}, nil
}

type sfxAdapter struct{ client *elevenlabs.Client }

func (a *sfxAdapter) Process(ctx context.Context, req *elevenlabs.SoundEffectRequest) (*port.JobResult, error) {
	result, err := a.client.GenerateSoundEffect(ctx, req)
	if err != nil {
		return nil, err
	}
	return &port.JobResult{
		ResultURL:  result.AudioURL,
		ResultData: map[string]any{"audio_url": result.AudioURL, "duration": result.Duration},
	}, nil
}

type musicAdapter struct{ client *elevenlabs.Client }

func (a *musicAdapter) Process(ctx context.Context, req *elevenlabs.MusicRequest) (*port.JobResult, error) {
	result, err := a.client.GenerateMusic(ctx, req)
	if err != nil {
		return nil, err
	}
	return &port.JobResult{
		ResultURL:  result.AudioURL,
		ResultData: map[string]any{"audio_url": result.AudioURL, "duration": result.Duration},
	}, nil
}

// SoundService encapsulates ElevenLabs TTS, SFX, music, and voice listing.
type SoundService struct {
	jobManager     *jobs.Manager
	client         *elevenlabs.Client
	ttsProcessor   pkgservice.JobProcessor[elevenlabs.TTSRequest]
	sfxProcessor   pkgservice.JobProcessor[elevenlabs.SoundEffectRequest]
	musicProcessor pkgservice.JobProcessor[elevenlabs.MusicRequest]
}

func NewSoundService(cfg *config.Config, jobManager *jobs.Manager) (*SoundService, error) {
	apiKey, err := cfg.GetAPIKey("elevenlabs")
	if err != nil {
		return nil, fmt.Errorf("ElevenLabs API key not configured: %w", err)
	}
	httpClient := common.NewHTTPClient(
		time.Duration(cfg.HTTPTimeout)*time.Second,
		cfg.HTTPRetryMax,
		time.Duration(cfg.HTTPRetryWait)*time.Second,
	)
	client := elevenlabs.NewClient(apiKey, httpClient, cfg.OutputDir+"/elevenlabs", cfg.ElevenLabsBaseURL)
	return &SoundService{
		jobManager:     jobManager,
		client:         client,
		ttsProcessor:   pkgservice.NewJobProcessor(jobManager, &ttsAdapter{client: client}),
		sfxProcessor:   pkgservice.NewJobProcessor(jobManager, &sfxAdapter{client: client}),
		musicProcessor: pkgservice.NewJobProcessor(jobManager, &musicAdapter{client: client}),
	}, nil
}

// GenerateSpeech creates a job and starts async TTS generation. Returns the job ID.
func (s *SoundService) GenerateSpeech(ctx context.Context, req *elevenlabs.TTSRequest) (string, error) {
	jobID, err := s.jobManager.CreateJob(ctx, types.ProviderElevenLabs, map[string]any{
		"text": req.Text, "voice_id": req.VoiceID,
	})
	if err != nil {
		return "", err
	}
	go s.ttsProcessor.Run(ctx, jobID, req)
	return jobID, nil
}

// GenerateSoundEffects creates a job and starts async SFX generation. Returns the job ID.
func (s *SoundService) GenerateSoundEffects(ctx context.Context, req *elevenlabs.SoundEffectRequest) (string, error) {
	jobID, err := s.jobManager.CreateJob(ctx, types.ProviderElevenLabs, map[string]any{
		"text": req.Text,
	})
	if err != nil {
		return "", err
	}
	go s.sfxProcessor.Run(ctx, jobID, req)
	return jobID, nil
}

// GenerateMusic creates a job and starts async music generation. Returns the job ID.
func (s *SoundService) GenerateMusic(ctx context.Context, req *elevenlabs.MusicRequest) (string, error) {
	jobID, err := s.jobManager.CreateJob(ctx, types.ProviderElevenLabs, map[string]any{
		"text": req.Text,
	})
	if err != nil {
		return "", err
	}
	go s.musicProcessor.Run(ctx, jobID, req)
	return jobID, nil
}

// GetVoices returns available ElevenLabs voices (synchronous).
func (s *SoundService) GetVoices(ctx context.Context, opts *elevenlabs.GetVoicesOptions) (*elevenlabs.VoicesResponse, error) {
	return s.client.GetVoices(ctx, opts)
}
