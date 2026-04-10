package grok

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/providers"
	"github.com/bryanperez/laguna-escondida-marketing/backend/pkg/common"
)

const (
	pollInterval = 5 * time.Second
	maxPollTime  = 10 * time.Minute
)

// Client is the xAI Grok text-to-video generation client.
type Client struct {
	baseClient
}

// NewClient creates a new Grok client.
// If baseURL is empty, the default xAI API URL is used.
func NewClient(apiKey string, httpClient *common.HTTPClient, outputDir string, baseURL string) *Client {
	return &Client{
		baseClient: newBaseClient(apiKey, httpClient, outputDir, baseURL),
	}
}

// ProviderName returns the provider identifier.
func (c *Client) ProviderName() string { return "grok" }

// GenerateVideo generates a video from an image using xAI Grok.
// Implements providers.ImageToVideoProvider.
func (c *Client) GenerateVideo(ctx context.Context, imageURL string, prompt string, opts providers.ImageToVideoOptions) (*providers.VideoResult, error) {
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
	resolution := opts.Mode // reuse Mode field for resolution ("480p" / "720p")
	if resolution == "" {
		resolution = DefaultResolution
	}

	req := &GenerateRequest{
		SourceImage: imageURL,
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

	videoPath, err := c.DownloadVideo(ctx, result.Video.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to download video: %w", err)
	}

	return &providers.VideoResult{
		LocalPath:   videoPath,
		ProviderURL: result.Video.URL,
		MimeType:    "video/mp4",
	}, nil
}

// submit sends an image-to-video request to xAI and returns the generation ID.
func (c *Client) submit(ctx context.Context, req *GenerateRequest) (*SubmitResponse, error) {
	body := map[string]any{
		"model":        ModelID,
		"prompt":       req.Prompt,
		"duration":     req.Duration,
		"aspect_ratio": req.AspectRatio,
		"resolution":   req.Resolution,
	}
	if req.SourceImage != "" {
		body["source_image"] = req.SourceImage
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
			Provider:   "grok",
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

// DownloadVideo downloads and saves the video from the completed generation.
func (c *Client) DownloadVideo(ctx context.Context, videoURL string) (string, error) {
	return c.downloadVideo(ctx, videoURL)
}
