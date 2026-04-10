package grok

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/providers"
	"github.com/bryanperez/laguna-escondida-marketing/backend/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestHTTPClient() *common.HTTPClient {
	return common.NewHTTPClient(10*time.Second, 0, 0)
}

// --- Constructor & basic ---

func TestNewClient_DefaultBaseURL(t *testing.T) {
	client := NewClient("key", newTestHTTPClient(), t.TempDir(), "")
	assert.Equal(t, DefaultXAIBaseURL, client.baseURL)
}

func TestNewClient_CustomBaseURL(t *testing.T) {
	client := NewClient("key", newTestHTTPClient(), t.TempDir(), "https://custom.api")
	assert.Equal(t, "https://custom.api", client.baseURL)
}

func TestProviderName(t *testing.T) {
	client := NewClient("key", newTestHTTPClient(), t.TempDir(), "")
	assert.Equal(t, "grok", client.ProviderName())
}

// --- submit ---

func TestSubmit_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "/videos/generations", r.URL.Path)

		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, ModelID, body["model"])
		assert.Equal(t, "A test prompt", body["prompt"])
		assert.Equal(t, float64(DefaultDuration), body["duration"])
		assert.Equal(t, DefaultAspectRatio, body["aspect_ratio"])
		assert.Equal(t, DefaultResolution, body["resolution"])
		assert.Equal(t, "https://example.com/image.jpg", body["source_image"])

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(SubmitResponse{RequestID: "gen-123"})
	}))
	defer server.Close()

	client := NewClient("test-key", newTestHTTPClient(), t.TempDir(), server.URL)
	resp, err := client.submit(context.Background(), &GenerateRequest{
		SourceImage: "https://example.com/image.jpg",
		Prompt:      "A test prompt",
		Duration:    DefaultDuration,
		AspectRatio: DefaultAspectRatio,
		Resolution:  DefaultResolution,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "gen-123", resp.RequestID)
}

func TestSubmit_WithoutSourceImage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))

		_, hasSourceImage := body["source_image"]
		assert.False(t, hasSourceImage, "source_image should be omitted when empty")

		json.NewEncoder(w).Encode(SubmitResponse{RequestID: "gen-456"})
	}))
	defer server.Close()

	client := NewClient("test-key", newTestHTTPClient(), t.TempDir(), server.URL)
	resp, err := client.submit(context.Background(), &GenerateRequest{
		Prompt:      "text-only prompt",
		Duration:    DefaultDuration,
		AspectRatio: DefaultAspectRatio,
		Resolution:  DefaultResolution,
	})

	require.NoError(t, err)
	assert.Equal(t, "gen-456", resp.RequestID)
}

func TestSubmit_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid_api_key"}`))
	}))
	defer server.Close()

	client := NewClient("bad-key", newTestHTTPClient(), t.TempDir(), server.URL)
	_, err := client.submit(context.Background(), &GenerateRequest{
		Prompt:      "test",
		Duration:    DefaultDuration,
		AspectRatio: DefaultAspectRatio,
		Resolution:  DefaultResolution,
	})

	require.Error(t, err)
	var apiErr *common.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusUnauthorized, apiErr.StatusCode)
	assert.Equal(t, "grok", apiErr.Provider)
}

// --- getStatus ---

func TestGetStatus_Pending(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Contains(t, r.URL.Path, "gen-123")

		json.NewEncoder(w).Encode(GenerateResponse{ Status: StatusPending})
	}))
	defer server.Close()

	client := NewClient("test-key", newTestHTTPClient(), t.TempDir(), server.URL)
	resp, err := client.getStatus(context.Background(), "gen-123")

	require.NoError(t, err)
	assert.Equal(t, StatusPending, resp.Status)
	assert.Nil(t, resp.Video)
}

func TestGetStatus_Done(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(GenerateResponse{
			Status: StatusDone,
			Video:  &GrokVideo{URL: "https://cdn.x.ai/video.mp4", Duration: 5},
		})
	}))
	defer server.Close()

	client := NewClient("test-key", newTestHTTPClient(), t.TempDir(), server.URL)
	resp, err := client.getStatus(context.Background(), "gen-123")

	require.NoError(t, err)
	assert.Equal(t, StatusDone, resp.Status)
	require.NotNil(t, resp.Video)
	assert.Equal(t, "https://cdn.x.ai/video.mp4", resp.Video.URL)
}

func TestGetStatus_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"generation_not_found"}`))
	}))
	defer server.Close()

	client := NewClient("test-key", newTestHTTPClient(), t.TempDir(), server.URL)
	_, err := client.getStatus(context.Background(), "nonexistent")

	require.Error(t, err)
	var apiErr *common.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.StatusCode)
	assert.Equal(t, "grok", apiErr.Provider)
}

// --- DownloadVideo ---

func TestDownloadVideo_WritesFileToDisk(t *testing.T) {
	fakeMP4 := []byte{0x00, 0x00, 0x00, 0x18, 0x66, 0x74, 0x79, 0x70} // ftyp box

	videoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "video/mp4")
		w.Write(fakeMP4)
	}))
	defer videoServer.Close()

	outputDir := t.TempDir()
	client := NewClient("test-key", newTestHTTPClient(), outputDir, "")

	path, err := client.DownloadVideo(context.Background(), videoServer.URL+"/video.mp4")

	require.NoError(t, err)
	assert.True(t, filepath.IsAbs(path))
	assert.Contains(t, path, outputDir)
	assert.Equal(t, ".mp4", filepath.Ext(path))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, fakeMP4, data)
}

func TestDownloadVideo_CreatesOutputDir(t *testing.T) {
	videoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("fake-video-data"))
	}))
	defer videoServer.Close()

	outputDir := filepath.Join(t.TempDir(), "new", "subdir")
	client := NewClient("test-key", newTestHTTPClient(), outputDir, "")

	path, err := client.DownloadVideo(context.Background(), videoServer.URL+"/video.mp4")

	require.NoError(t, err)
	assert.Contains(t, path, outputDir)
	assert.DirExists(t, outputDir)
}

func TestDownloadVideo_APIError(t *testing.T) {
	videoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer videoServer.Close()

	client := NewClient("test-key", newTestHTTPClient(), t.TempDir(), "")
	_, err := client.DownloadVideo(context.Background(), videoServer.URL+"/video.mp4")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to download video")
}

// --- pollStatus ---
// NOTE: pollStatus uses a 5-second ticker (pollInterval constant), so tests that
// wait for at least one tick will take ≥5 s. This is expected.

func TestPollStatus_Timeout(t *testing.T) {
	// Context expires in 1s — before the first 5s tick fires.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(GenerateResponse{ Status: StatusPending})
	}))
	defer server.Close()

	client := NewClient("test-key", newTestHTTPClient(), t.TempDir(), server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, err := client.pollStatus(ctx, "gen-123", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "polling timeout")
}

func TestPollStatus_Failed(t *testing.T) {
	// Returns "failed" on the first tick (~5 s wait).
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(GenerateResponse{ Status: StatusFailed})
	}))
	defer server.Close()

	client := NewClient("test-key", newTestHTTPClient(), t.TempDir(), server.URL)

	_, err := client.pollStatus(context.Background(), "gen-123", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "generation failed")
}

func TestPollStatus_FailedWithErrorDetail(t *testing.T) {
	// Returns "failed" with a structured error object (~5 s wait).
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"failed","error":{"code":"content_moderation","message":"Video rejected by content moderation"}}`))
	}))
	defer server.Close()

	client := NewClient("test-key", newTestHTTPClient(), t.TempDir(), server.URL)

	_, err := client.pollStatus(context.Background(), "gen-123", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "generation failed: Video rejected by content moderation")
}

func TestPollStatus_Expired(t *testing.T) {
	// Returns "expired" on the first tick (~5 s wait).
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(GenerateResponse{ Status: StatusExpired})
	}))
	defer server.Close()

	client := NewClient("test-key", newTestHTTPClient(), t.TempDir(), server.URL)

	_, err := client.pollStatus(context.Background(), "gen-123", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "generation expired")
}

func TestPollStatus_PendingThenDone(t *testing.T) {
	// First tick: pending. Second tick: done. (~10 s wait)
	var callCount atomic.Int32
	var videoURL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		if n == 1 {
			json.NewEncoder(w).Encode(GenerateResponse{ Status: StatusPending})
			return
		}
		json.NewEncoder(w).Encode(GenerateResponse{
			Status: StatusDone,
			Video:  &GrokVideo{URL: videoURL, Duration: 5},
		})
	}))
	defer server.Close()
	videoURL = server.URL + "/video.mp4"

	client := NewClient("test-key", newTestHTTPClient(), t.TempDir(), server.URL)

	result, err := client.pollStatus(context.Background(), "gen-123", nil)
	require.NoError(t, err)
	require.NotNil(t, result.Video)
	assert.Equal(t, StatusDone, result.Status)
	assert.Equal(t, int32(2), callCount.Load(), "should poll exactly twice")
}

func TestPollStatus_ProgressCallback(t *testing.T) {
	// First tick: pending (callback called with progress < 1). Second tick: done (callback called with 1.0).
	var callCount atomic.Int32
	var videoURL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		if n == 1 {
			json.NewEncoder(w).Encode(GenerateResponse{ Status: StatusPending})
			return
		}
		json.NewEncoder(w).Encode(GenerateResponse{
			Status: StatusDone,
			Video:  &GrokVideo{URL: videoURL},
		})
	}))
	defer server.Close()
	videoURL = server.URL + "/video.mp4"

	client := NewClient("test-key", newTestHTTPClient(), t.TempDir(), server.URL)

	var progressValues []float64
	result, err := client.pollStatus(context.Background(), "gen-123", func(p float64) {
		progressValues = append(progressValues, p)
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, progressValues, "progress callback should have been called")
	assert.Equal(t, 1.0, progressValues[len(progressValues)-1], "last progress value should be 1.0")
}

// --- GenerateVideo (end-to-end) ---

func TestGenerateVideo_Success(t *testing.T) {
	// submit → 1 poll (done) → download. (~5 s wait for first tick)
	fakeMP4 := []byte{0x00, 0x00, 0x00, 0x18, 0x66, 0x74, 0x79, 0x70}

	videoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "video/mp4")
		w.Write(fakeMP4)
	}))
	defer videoServer.Close()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, ModelID, body["model"])
			assert.Equal(t, "A beautiful scene", body["prompt"])
			assert.Equal(t, float64(5), body["duration"])
			assert.Equal(t, "16:9", body["aspect_ratio"])
			assert.Equal(t, "480p", body["resolution"])
			assert.Equal(t, "https://example.com/img.jpg", body["source_image"])

			json.NewEncoder(w).Encode(SubmitResponse{RequestID: "gen-abc"})
			return
		}
		// First and only poll: immediately done.
		json.NewEncoder(w).Encode(GenerateResponse{
			Status: StatusDone,
			Video:  &GrokVideo{URL: videoServer.URL + "/video.mp4", Duration: 5},
		})
	}))
	defer apiServer.Close()

	outputDir := t.TempDir()
	client := NewClient("test-key", newTestHTTPClient(), outputDir, apiServer.URL)

	result, err := client.GenerateVideo(context.Background(), "https://example.com/img.jpg", "A beautiful scene", providers.ImageToVideoOptions{
		Duration:    "5",
		AspectRatio: "16:9",
		Mode:        "480p",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Contains(t, result.LocalPath, outputDir)
	assert.Equal(t, ".mp4", filepath.Ext(result.LocalPath))
	assert.Equal(t, videoServer.URL+"/video.mp4", result.ProviderURL)
	assert.Equal(t, "video/mp4", result.MimeType)

	data, err := os.ReadFile(result.LocalPath)
	require.NoError(t, err)
	assert.Equal(t, fakeMP4, data)
}

func TestGenerateVideo_DefaultOptions(t *testing.T) {
	// Verifies defaults (duration=5, aspect_ratio=16:9, resolution=480p) are applied when opts are empty.
	// Uses a short context so polling times out before the first tick — we only care about the submit.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, float64(DefaultDuration), body["duration"])
		assert.Equal(t, DefaultAspectRatio, body["aspect_ratio"])
		assert.Equal(t, DefaultResolution, body["resolution"])

		json.NewEncoder(w).Encode(SubmitResponse{RequestID: "gen-def"})
	}))
	defer server.Close()

	client := NewClient("test-key", newTestHTTPClient(), t.TempDir(), server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, _ = client.GenerateVideo(ctx, "", "test", providers.ImageToVideoOptions{})
}

func TestGenerateVideo_SubmitAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"quota_exceeded"}`))
	}))
	defer server.Close()

	client := NewClient("test-key", newTestHTTPClient(), t.TempDir(), server.URL)
	result, err := client.GenerateVideo(context.Background(), "", "test", providers.ImageToVideoOptions{})

	require.Error(t, err)
	assert.Nil(t, result)
	var apiErr *common.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusForbidden, apiErr.StatusCode)
}
