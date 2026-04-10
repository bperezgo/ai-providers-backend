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

// ReferenceVideoClient generates videos influenced by reference images.
// Reference images affect the visual style without locking the first frame.
// Uses the xAI "reference_images" field with <IMAGE_N> prompt placeholders.
type ReferenceVideoClient struct {
	baseClient
}

// NewReferenceVideoClient creates a new reference-image video Grok client.
// If baseURL is empty, the default xAI API URL is used.
func NewReferenceVideoClient(apiKey string, httpClient *common.HTTPClient, outputDir string, baseURL string) *ReferenceVideoClient {
	return &ReferenceVideoClient{
		baseClient: newBaseClient(apiKey, httpClient, outputDir, baseURL),
	}
}

// ProviderName returns the provider identifier.
func (c *ReferenceVideoClient) ProviderName() string { return "grok-ref" }

// GenerateVideoFromReferences generates a video influenced by reference images.
// The prompt should use <IMAGE_1>, <IMAGE_2>, etc. placeholders to reference each image.
// Implements providers.ReferenceImageVideoProvider.
func (c *ReferenceVideoClient) GenerateVideoFromReferences(ctx context.Context, prompt string, referenceImageURLs []string, opts providers.ReferenceImageVideoOptions) (*providers.VideoResult, error) {
	if len(referenceImageURLs) == 0 {
		return nil, fmt.Errorf("at least one reference image URL is required")
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

	refs := make([]ReferenceImage, len(referenceImageURLs))
	for i, url := range referenceImageURLs {
		refs[i] = ReferenceImage{URL: url}
	}

	req := &ReferenceVideoGenerateRequest{
		Prompt:          prompt,
		ReferenceImages: refs,
		Duration:        duration,
		AspectRatio:     aspectRatio,
		Resolution:      resolution,
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

// submit sends a reference-image video request to xAI using the "reference_images" field.
func (c *ReferenceVideoClient) submit(ctx context.Context, req *ReferenceVideoGenerateRequest) (*SubmitResponse, error) {
	body := map[string]any{
		"model":            ModelID,
		"prompt":           req.Prompt,
		"reference_images": req.ReferenceImages,
		"duration":         req.Duration,
		"aspect_ratio":     req.AspectRatio,
		"resolution":       req.Resolution,
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
			Provider:   "grok-ref",
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
