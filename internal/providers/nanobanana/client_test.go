package nanobanana

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bryanperez/laguna-escondida-marketing/backend/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestHTTPClient() *common.HTTPClient {
	return common.NewHTTPClient(10*time.Second, 0, time.Second)
}

func TestSubmit_success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Key test-api-key", r.Header.Get("Authorization"))

		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "a beautiful sunset", body["prompt"])

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(FalSubmitResponse{
			RequestID:   "req-123",
			ResponseURL: "https://fal.ai/response/req-123",
			StatusURL:   "https://fal.ai/status/req-123",
			CancelURL:   "https://fal.ai/cancel/req-123",
		})
	}))
	defer server.Close()

	// Override FalBaseURL for testing by creating a client that targets our mock
	client := &Client{
		apiKey:     "test-api-key",
		httpClient: newTestHTTPClient(),
		outputDir:  t.TempDir(),
	}

	// Build request manually to target mock server
	req := &GenerateRequest{Prompt: "a beautiful sunset"}
	resp, err := submitToURL(client, server.URL+"/"+ModelID, req)
	require.NoError(t, err)

	assert.Equal(t, "req-123", resp.RequestID)
	assert.Equal(t, "https://fal.ai/response/req-123", resp.ResponseURL)
	assert.Equal(t, "https://fal.ai/status/req-123", resp.StatusURL)
}

func TestSubmit_authError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "invalid API key"}`))
	}))
	defer server.Close()

	client := &Client{
		apiKey:     "bad-key",
		httpClient: newTestHTTPClient(),
		outputDir:  t.TempDir(),
	}

	req := &GenerateRequest{Prompt: "test"}
	_, err := submitToURL(client, server.URL+"/"+ModelID, req)
	require.Error(t, err)

	var apiErr *common.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusUnauthorized, apiErr.StatusCode)
}

func TestPollStatus_completedImmediately(t *testing.T) {
	statusServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(FalStatusResponse{Status: StatusCompleted})
	}))
	defer statusServer.Close()

	resultServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(FalResultResponse{
			Images: []FalImage{
				{URL: "https://cdn.fal.ai/image.png", Width: 1024, Height: 1024},
			},
		})
	}))
	defer resultServer.Close()

	client := &Client{
		apiKey:     "test-key",
		httpClient: newTestHTTPClient(),
		outputDir:  t.TempDir(),
	}

	result, err := client.PollStatus(context.Background(), statusServer.URL, resultServer.URL, nil)
	require.NoError(t, err)
	require.Len(t, result.Images, 1)
	assert.Equal(t, "https://cdn.fal.ai/image.png", result.Images[0].URL)
	assert.Equal(t, 1024, result.Images[0].Width)
}

func TestPollStatus_queueThenCompleted(t *testing.T) {
	var callCount atomic.Int32

	statusServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		if n <= 2 {
			json.NewEncoder(w).Encode(FalStatusResponse{Status: StatusInQueue})
			return
		}
		json.NewEncoder(w).Encode(FalStatusResponse{Status: StatusCompleted})
	}))
	defer statusServer.Close()

	var resultCalls atomic.Int32
	resultServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resultCalls.Add(1)
		json.NewEncoder(w).Encode(FalResultResponse{
			Images: []FalImage{{URL: "https://cdn.fal.ai/done.png", Width: 512, Height: 512}},
		})
	}))
	defer resultServer.Close()

	client := &Client{
		apiKey:     "test-key",
		httpClient: newTestHTTPClient(),
		outputDir:  t.TempDir(),
	}

	result, err := client.PollStatus(context.Background(), statusServer.URL, resultServer.URL, nil)
	require.NoError(t, err)
	require.Len(t, result.Images, 1)
	assert.Equal(t, "https://cdn.fal.ai/done.png", result.Images[0].URL)
	assert.Equal(t, int32(1), resultCalls.Load(), "getResult should only be called once on COMPLETED")
}

func TestPollStatus_timeout(t *testing.T) {
	statusServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(FalStatusResponse{Status: StatusInQueue})
	}))
	defer statusServer.Close()

	client := &Client{
		apiKey:     "test-key",
		httpClient: newTestHTTPClient(),
		outputDir:  t.TempDir(),
	}

	// Use a short context timeout to speed up the test
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.PollStatus(ctx, statusServer.URL, "http://unused", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "polling timeout")
}

func TestPollStatus_failed(t *testing.T) {
	statusServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(FalStatusResponse{Status: StatusFailed})
	}))
	defer statusServer.Close()

	client := &Client{
		apiKey:     "test-key",
		httpClient: newTestHTTPClient(),
		outputDir:  t.TempDir(),
	}

	_, err := client.PollStatus(context.Background(), statusServer.URL, "http://unused", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "generation failed")
}

func TestDownloadImage_writesFileToDisk(t *testing.T) {
	expectedBytes := []byte{0x89, 0x50, 0x4E, 0x47} // PNG magic bytes

	imageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(expectedBytes)
	}))
	defer imageServer.Close()

	outputDir := t.TempDir()
	client := &Client{
		apiKey:     "test-key",
		httpClient: newTestHTTPClient(),
		outputDir:  outputDir,
	}

	filePath, err := client.DownloadImage(context.Background(), imageServer.URL+"/image.png")
	require.NoError(t, err)

	// Verify file exists and has correct content
	assert.True(t, filepath.IsAbs(filePath))
	data, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, expectedBytes, data)
}

func TestPollStatus_progressCallback(t *testing.T) {
	var callCount atomic.Int32

	statusServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		if n <= 1 {
			json.NewEncoder(w).Encode(FalStatusResponse{Status: StatusInProgress})
			return
		}
		json.NewEncoder(w).Encode(FalStatusResponse{Status: StatusCompleted})
	}))
	defer statusServer.Close()

	resultServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(FalResultResponse{
			Images: []FalImage{{URL: "https://cdn.fal.ai/img.png", Width: 256, Height: 256}},
		})
	}))
	defer resultServer.Close()

	client := &Client{
		apiKey:     "test-key",
		httpClient: newTestHTTPClient(),
		outputDir:  t.TempDir(),
	}

	var progressValues []float64
	callback := func(p float64) {
		progressValues = append(progressValues, p)
	}

	result, err := client.PollStatus(context.Background(), statusServer.URL, resultServer.URL, callback)
	require.NoError(t, err)
	require.Len(t, result.Images, 1)
	require.NotEmpty(t, progressValues)

	// Last progress value should be 1.0 (completion)
	assert.Equal(t, 1.0, progressValues[len(progressValues)-1])
}

// submitToURL is a test helper that posts to a specific URL instead of FalBaseURL.
func submitToURL(c *Client, url string, req *GenerateRequest) (*FalSubmitResponse, error) {
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

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	ctx := context.Background()
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Key "+c.apiKey)

	resp, err := c.httpClient.DoWithRetry(ctx, httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, &common.APIError{
			Provider:   "nanobanana",
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
		}
	}

	var submitResp FalSubmitResponse
	if err := json.NewDecoder(resp.Body).Decode(&submitResp); err != nil {
		return nil, err
	}
	return &submitResp, nil
}
