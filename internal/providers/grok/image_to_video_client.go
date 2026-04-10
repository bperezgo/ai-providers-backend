package grok

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/providers"
	"github.com/bryanperez/laguna-escondida-marketing/backend/pkg/common"
)

// ImageToVideoClient generates videos where the source image becomes the first
// frame. It uses the xAI "image" field (distinct from the text-to-video
// "source_image" field).
type ImageToVideoClient struct {
	baseClient
}

// NewImageToVideoClient creates a new image-to-video Grok client.
// If baseURL is empty, the default xAI API URL is used.
func NewImageToVideoClient(apiKey string, httpClient *common.HTTPClient, outputDir string, baseURL string) *ImageToVideoClient {
	return &ImageToVideoClient{
		baseClient: newBaseClient(apiKey, httpClient, outputDir, baseURL),
	}
}

// ProviderName returns the provider identifier.
func (c *ImageToVideoClient) ProviderName() string { return "grok-i2v" }

// GenerateVideo generates a video from an image using xAI Grok.
// The source image becomes the first frame of the generated video.
// Implements providers.ImageToVideoProvider.
func (c *ImageToVideoClient) GenerateVideo(ctx context.Context, imageURL string, prompt string, opts providers.ImageToVideoOptions) (*providers.VideoResult, error) {
	if imageURL == "" {
		return nil, fmt.Errorf("imageURL is required for image-to-video generation")
	}

	duration := DefaultDuration
	if opts.Duration != "" {
		var d int
		if _, err := fmt.Sscanf(opts.Duration, "%d", &d); err == nil && d > 0 {
			duration = d
		}
	}
	aspectRatio := opts.AspectRatio
	if aspectRatio == "" {
		aspectRatio = DefaultAspectRatio
	}
	resolution := opts.Mode
	if resolution == "" {
		resolution = DefaultResolution
	}

	req := &ImageToVideoGenerateRequest{
		Image:       ImageURL{URL: imageURL},
		Prompt:      prompt,
		Duration:    duration,
		AspectRatio: aspectRatio,
		Resolution:  resolution,
	}

	resp, err := c.submit(ctx, req)
	if err != nil {
		return nil, err
	}

	result, err := c.pollStatus(ctx, resp.RequestID, nil)
	if err != nil {
		return nil, err
	}

	if result.Video == nil {
		return nil, fmt.Errorf("no video in completion response")
	}

	videoPath, err := c.downloadVideo(ctx, result.Video.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to download video: %w", err)
	}

	return &providers.VideoResult{
		LocalPath:   videoPath,
		ProviderURL: result.Video.URL,
		MimeType:    "video/mp4",
	}, nil
}

// submit sends an image-to-video request to xAI using the "image" field.
func (c *ImageToVideoClient) submit(ctx context.Context, req *ImageToVideoGenerateRequest) (*SubmitResponse, error) {
	body := map[string]any{
		"model":        ModelID,
		"image":        map[string]string{"url": req.Image.URL},
		"duration":     req.Duration,
		"aspect_ratio": req.AspectRatio,
		"resolution":   req.Resolution,
	}
	if req.Prompt != "" {
		body["prompt"] = req.Prompt
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/videos/generations", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.DoWithRetry(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, &common.APIError{
			Provider:   "grok-i2v",
			StatusCode: resp.StatusCode,
			Message:    string(bodyBytes),
		}
	}

	var submitResp SubmitResponse
	if err := json.NewDecoder(resp.Body).Decode(&submitResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &submitResp, nil
}
