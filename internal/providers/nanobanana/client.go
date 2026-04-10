package nanobanana

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/providers"
	"github.com/bryanperez/laguna-escondida-marketing/backend/pkg/common"
	"github.com/google/uuid"
)

const (
	pollInterval = 3 * time.Second
	maxPollTime  = 3 * time.Minute
)

// Client is the Nano Banana Pro API client via Fal.ai
type Client struct {
	apiKey     string
	httpClient *common.HTTPClient
	outputDir  string
	baseURL    string
}

// NewClient creates a new Nano Banana client.
// If baseURL is empty, the default Fal.ai URL is used.
func NewClient(apiKey string, httpClient *common.HTTPClient, outputDir string, baseURL string) *Client {
	if baseURL == "" {
		baseURL = DefaultFalBaseURL
	}
	return &Client{
		apiKey:     apiKey,
		httpClient: httpClient,
		outputDir:  outputDir,
		baseURL:    baseURL,
	}
}

// ProviderName returns the provider identifier.
func (c *Client) ProviderName() string { return "nanobanana" }

// GenerateImage generates an image from a text prompt, handling the full Fal.ai queue flow.
// Implements providers.TextToImageProvider.
func (c *Client) GenerateImage(ctx context.Context, prompt string, opts providers.TextToImageOptions) (*providers.ImageResult, error) {
	imageSize := opts.Size
	if imageSize == "" {
		imageSize = DefaultImageSize
	}
	n := opts.N
	if n <= 0 {
		n = DefaultNumImages
	}

	req := &GenerateRequest{
		Prompt:            prompt,
		ImageSize:         imageSize,
		NumImages:         n,
		NumInferenceSteps: DefaultNumInferenceSteps,
		GuidanceScale:     DefaultGuidanceScale,
	}

	submitResp, err := c.Submit(ctx, req)
	if err != nil {
		return nil, err
	}

	pollResult, err := c.PollStatus(ctx, submitResp.StatusURL, submitResp.ResponseURL, nil)
	if err != nil {
		return nil, err
	}

	if len(pollResult.Images) == 0 {
		return nil, fmt.Errorf("no images returned")
	}

	img := pollResult.Images[0]
	imagePath, err := c.DownloadImage(ctx, img.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}

	width, height := img.Width, img.Height
	if width == 0 || height == 0 {
		w, h, err := imageDimensions(imagePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read image dimensions: %w", err)
		}
		width, height = w, h
	}

	return &providers.ImageResult{
		LocalPath:   imagePath,
		ProviderURL: img.URL,
		Width:       width,
		Height:      height,
		MimeType:    "image/png",
	}, nil
}

// Submit submits a text-to-image request to Nano Banana Pro via Fal.ai queue.
func (c *Client) Submit(ctx context.Context, req *GenerateRequest) (*FalSubmitResponse, error) {
	if req.ImageSize == "" {
		req.ImageSize = DefaultImageSize
	}
	if req.NumImages <= 0 {
		req.NumImages = DefaultNumImages
	}
	if req.NumInferenceSteps <= 0 {
		req.NumInferenceSteps = DefaultNumInferenceSteps
	}
	if req.GuidanceScale == 0 {
		req.GuidanceScale = DefaultGuidanceScale
	}

	body := map[string]any{
		"prompt":              req.Prompt,
		"image_size":          req.ImageSize,
		"num_images":          req.NumImages,
		"num_inference_steps": req.NumInferenceSteps,
		"guidance_scale":      req.GuidanceScale,
	}
	if req.NegativePrompt != "" {
		body["negative_prompt"] = req.NegativePrompt
	}
	if req.Seed != 0 {
		body["seed"] = req.Seed
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/%s", c.baseURL, ModelID)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Key "+c.apiKey)

	resp, err := c.httpClient.DoWithRetry(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, &common.APIError{
			Provider:   "nanobanana",
			StatusCode: resp.StatusCode,
			Message:    string(bodyBytes),
		}
	}

	var submitResp FalSubmitResponse
	if err := json.NewDecoder(resp.Body).Decode(&submitResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &submitResp, nil
}

// PollStatus polls the Fal.ai queue until the job completes or times out
func (c *Client) PollStatus(ctx context.Context, statusURL, responseURL string, progressCallback func(float64)) (*FalResultResponse, error) {
	pollCtx, cancel := context.WithTimeout(ctx, maxPollTime)
	defer cancel()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	attempt := 0
	for {
		select {
		case <-pollCtx.Done():
			return nil, fmt.Errorf("polling timeout after %v", maxPollTime)
		case <-ticker.C:
			attempt++
			status, err := c.getStatus(pollCtx, statusURL)
			if err != nil {
				return nil, fmt.Errorf("failed to get status: %w", err)
			}

			switch status.Status {
			case StatusCompleted:
				if progressCallback != nil {
					progressCallback(1.0)
				}
				return c.getResult(pollCtx, responseURL)
			case StatusFailed:
				return nil, fmt.Errorf("generation failed")
			case StatusInQueue, StatusInProgress:
				if progressCallback != nil {
					// Image gen is fast; 60 attempts at 3s = 3 min max
					progress := float64(attempt) / 20.0
					if progress > 0.9 {
						progress = 0.9
					}
					progressCallback(progress)
				}
				continue
			default:
				continue
			}
		}
	}
}

// getStatus retrieves the current queue status
func (c *Client) getStatus(ctx context.Context, url string) (*FalStatusResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Key "+c.apiKey)

	resp, err := c.httpClient.DoWithRetry(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, &common.APIError{
			Provider:   "nanobanana",
			StatusCode: resp.StatusCode,
			Message:    string(bodyBytes),
		}
	}

	var statusResp FalStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		return nil, fmt.Errorf("failed to decode status response: %w", err)
	}

	return &statusResp, nil
}

// getResult retrieves the completed generation result
func (c *Client) getResult(ctx context.Context, url string) (*FalResultResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Key "+c.apiKey)

	resp, err := c.httpClient.DoWithRetry(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, &common.APIError{
			Provider:   "nanobanana",
			StatusCode: resp.StatusCode,
			Message:    string(bodyBytes),
		}
	}

	var resultResp FalResultResponse
	if err := json.NewDecoder(resp.Body).Decode(&resultResp); err != nil {
		return nil, fmt.Errorf("failed to decode result response: %w", err)
	}

	if resultResp.Error != "" {
		return nil, fmt.Errorf("generation error: %s", resultResp.Error)
	}

	return &resultResp, nil
}

// DownloadImage downloads and saves the first image from the completed generation
func (c *Client) DownloadImage(ctx context.Context, imageURL string) (string, error) {
	if err := os.MkdirAll(c.outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", imageURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create download request: %w", err)
	}

	resp, err := c.httpClient.DoWithRetry(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download image: status %d", resp.StatusCode)
	}

	filename := fmt.Sprintf("nanobanana_%s_%d.png", uuid.Must(uuid.NewV7()).String(), time.Now().Unix())
	filePath := filepath.Join(c.outputDir, filename)

	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return "", fmt.Errorf("failed to write image data: %w", err)
	}

	return filePath, nil
}

// imageDimensions reads width and height from an image file on disk.
func imageDimensions(path string) (int, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0, err
	}
	return cfg.Width, cfg.Height, nil
}
