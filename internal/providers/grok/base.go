package grok

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/bryanperez/laguna-escondida-marketing/backend/pkg/common"
	"github.com/google/uuid"
)

// baseClient contains the shared xAI Grok infrastructure: polling, status
// checking, and video downloading. Concrete clients embed this struct.
type baseClient struct {
	apiKey     string
	httpClient *common.HTTPClient
	outputDir  string
	baseURL    string
}

func newBaseClient(apiKey string, httpClient *common.HTTPClient, outputDir string, baseURL string) baseClient {
	if baseURL == "" {
		baseURL = DefaultXAIBaseURL
	}
	return baseClient{
		apiKey:     apiKey,
		httpClient: httpClient,
		outputDir:  outputDir,
		baseURL:    baseURL,
	}
}

// pollStatus polls GET /v1/videos/{id} until the job completes or times out.
func (b *baseClient) pollStatus(ctx context.Context, id string, progressCallback func(float64)) (*GenerateResponse, error) {
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
			result, err := b.getStatus(pollCtx, id)
			if err != nil {
				return nil, fmt.Errorf("failed to get status: %w", err)
			}

			switch result.Status {
			case StatusDone:
				if progressCallback != nil {
					progressCallback(1.0)
				}
				return result, nil
			case StatusFailed:
				if result.Error != nil && result.Error.Message != "" {
					return nil, fmt.Errorf("generation failed: %s", result.Error.Message)
				}
				return nil, fmt.Errorf("generation failed")
			case StatusExpired:
				return nil, fmt.Errorf("generation expired")
			case StatusPending:
				if progressCallback != nil {
					progress := float64(attempt) / 60.0
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

// getStatus retrieves the current generation status from xAI.
func (b *baseClient) getStatus(ctx context.Context, id string) (*GenerateResponse, error) {
	url := fmt.Sprintf("%s/videos/%s", b.baseURL, id)
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+b.apiKey)

	resp, err := b.httpClient.DoWithRetry(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, &common.APIError{
			Provider:   "grok",
			StatusCode: resp.StatusCode,
			Message:    string(bodyBytes),
		}
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read status response: %w", err)
	}

	var genResp GenerateResponse
	if err := json.Unmarshal(bodyBytes, &genResp); err != nil {
		return nil, fmt.Errorf("failed to decode status response: %w", err)
	}

	return &genResp, nil
}

// downloadVideo downloads and saves the video from the completed generation.
func (b *baseClient) downloadVideo(ctx context.Context, videoURL string) (string, error) {
	if err := os.MkdirAll(b.outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", videoURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create download request: %w", err)
	}

	resp, err := b.httpClient.DoWithRetry(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to download video: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download video: status %d", resp.StatusCode)
	}

	filename := fmt.Sprintf("grok_%s_%d.mp4", uuid.Must(uuid.NewV7()).String(), time.Now().Unix())
	filePath := filepath.Join(b.outputDir, filename)

	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return "", fmt.Errorf("failed to write video data: %w", err)
	}

	return filePath, nil
}
