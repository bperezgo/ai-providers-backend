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

func newI2VTestHTTPClient() *common.HTTPClient {
	return common.NewHTTPClient(10*time.Second, 0, 0)
}

// --- Constructor & basic ---

func TestNewImageToVideoClient_DefaultBaseURL(t *testing.T) {
	client := NewImageToVideoClient("key", newI2VTestHTTPClient(), t.TempDir(), "")
	assert.Equal(t, DefaultXAIBaseURL, client.baseURL)
}

func TestNewImageToVideoClient_CustomBaseURL(t *testing.T) {
	client := NewImageToVideoClient("key", newI2VTestHTTPClient(), t.TempDir(), "https://custom.api")
	assert.Equal(t, "https://custom.api", client.baseURL)
}

func TestImageToVideoClient_ProviderName(t *testing.T) {
	client := NewImageToVideoClient("key", newI2VTestHTTPClient(), t.TempDir(), "")
	assert.Equal(t, "grok-i2v", client.ProviderName())
}

// --- submit ---

func TestImageToVideoClient_Submit_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "/videos/generations", r.URL.Path)

		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, ModelID, body["model"])
		imageObj, ok := body["image"].(map[string]any)
		require.True(t, ok, "image should be an object")
		assert.Equal(t, "https://example.com/photo.jpg", imageObj["url"])
		assert.Equal(t, "Animate this image", body["prompt"])
		assert.Equal(t, float64(DefaultDuration), body["duration"])
		assert.Equal(t, DefaultAspectRatio, body["aspect_ratio"])
		assert.Equal(t, DefaultResolution, body["resolution"])

		// Must NOT contain source_image or reference_images
		_, hasSourceImage := body["source_image"]
		assert.False(t, hasSourceImage)
		_, hasRefImages := body["reference_images"]
		assert.False(t, hasRefImages)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(SubmitResponse{RequestID: "i2v-123"})
	}))
	defer server.Close()

	client := NewImageToVideoClient("test-key", newI2VTestHTTPClient(), t.TempDir(), server.URL)
	resp, err := client.submit(context.Background(), &ImageToVideoGenerateRequest{
		Image:       ImageURL{URL: "https://example.com/photo.jpg"},
		Prompt:      "Animate this image",
		Duration:    DefaultDuration,
		AspectRatio: DefaultAspectRatio,
		Resolution:  DefaultResolution,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "i2v-123", resp.RequestID)
}

func TestImageToVideoClient_Submit_WithoutPrompt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))

		_, hasPrompt := body["prompt"]
		assert.False(t, hasPrompt, "prompt should be omitted when empty")
		imageObj, ok := body["image"].(map[string]any)
		require.True(t, ok, "image should be an object")
		assert.Equal(t, "https://example.com/photo.jpg", imageObj["url"])

		json.NewEncoder(w).Encode(SubmitResponse{RequestID: "i2v-456"})
	}))
	defer server.Close()

	client := NewImageToVideoClient("test-key", newI2VTestHTTPClient(), t.TempDir(), server.URL)
	resp, err := client.submit(context.Background(), &ImageToVideoGenerateRequest{
		Image:       ImageURL{URL: "https://example.com/photo.jpg"},
		Duration:    DefaultDuration,
		AspectRatio: DefaultAspectRatio,
		Resolution:  DefaultResolution,
	})

	require.NoError(t, err)
	assert.Equal(t, "i2v-456", resp.RequestID)
}

func TestImageToVideoClient_Submit_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid_api_key"}`))
	}))
	defer server.Close()

	client := NewImageToVideoClient("bad-key", newI2VTestHTTPClient(), t.TempDir(), server.URL)
	_, err := client.submit(context.Background(), &ImageToVideoGenerateRequest{
		Image:    ImageURL{URL: "https://example.com/photo.jpg"},
		Duration: DefaultDuration,
	})

	require.Error(t, err)
	var apiErr *common.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusUnauthorized, apiErr.StatusCode)
	assert.Equal(t, "grok-i2v", apiErr.Provider)
}

// --- GenerateVideo ---

func TestImageToVideoClient_GenerateVideo_RequiresImageURL(t *testing.T) {
	client := NewImageToVideoClient("key", newI2VTestHTTPClient(), t.TempDir(), "")
	_, err := client.GenerateVideo(context.Background(), "", "prompt", providers.ImageToVideoOptions{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "imageURL is required")
}

func TestImageToVideoClient_GenerateVideo_Success(t *testing.T) {
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
			assert.Equal(t, "https://example.com/img.jpg", body["image"])
			assert.Equal(t, "Animate the sunset", body["prompt"])
			assert.Equal(t, float64(10), body["duration"])
			assert.Equal(t, "9:16", body["aspect_ratio"])
			assert.Equal(t, "720p", body["resolution"])

			json.NewEncoder(w).Encode(SubmitResponse{RequestID: "i2v-abc"})
			return
		}
		json.NewEncoder(w).Encode(GenerateResponse{
			Status: StatusDone,
			Video:  &GrokVideo{URL: videoServer.URL + "/video.mp4", Duration: 10},
		})
	}))
	defer apiServer.Close()

	outputDir := t.TempDir()
	client := NewImageToVideoClient("test-key", newI2VTestHTTPClient(), outputDir, apiServer.URL)

	result, err := client.GenerateVideo(context.Background(), "https://example.com/img.jpg", "Animate the sunset", providers.ImageToVideoOptions{
		Duration:    "10",
		AspectRatio: "9:16",
		Mode:        "720p",
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

func TestImageToVideoClient_GenerateVideo_DefaultOptions(t *testing.T) {
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

		json.NewEncoder(w).Encode(SubmitResponse{RequestID: "i2v-def"})
	}))
	defer server.Close()

	client := NewImageToVideoClient("test-key", newI2VTestHTTPClient(), t.TempDir(), server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, _ = client.GenerateVideo(ctx, "https://example.com/img.jpg", "test", providers.ImageToVideoOptions{})
}

func TestImageToVideoClient_GenerateVideo_SubmitAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"quota_exceeded"}`))
	}))
	defer server.Close()

	client := NewImageToVideoClient("test-key", newI2VTestHTTPClient(), t.TempDir(), server.URL)
	result, err := client.GenerateVideo(context.Background(), "https://example.com/img.jpg", "test", providers.ImageToVideoOptions{})

	require.Error(t, err)
	assert.Nil(t, result)
	var apiErr *common.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusForbidden, apiErr.StatusCode)
}

func TestImageToVideoClient_GenerateVideo_PollPendingThenDone(t *testing.T) {
	fakeMP4 := []byte{0x00, 0x00, 0x00, 0x18, 0x66, 0x74, 0x79, 0x70}
	var callCount atomic.Int32

	videoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(fakeMP4)
	}))
	defer videoServer.Close()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			json.NewEncoder(w).Encode(SubmitResponse{RequestID: "i2v-poll"})
			return
		}
		n := callCount.Add(1)
		if n == 1 {
			json.NewEncoder(w).Encode(GenerateResponse{Status: StatusPending})
			return
		}
		json.NewEncoder(w).Encode(GenerateResponse{
			Status: StatusDone,
			Video:  &GrokVideo{URL: videoServer.URL + "/video.mp4", Duration: 5},
		})
	}))
	defer apiServer.Close()

	client := NewImageToVideoClient("test-key", newI2VTestHTTPClient(), t.TempDir(), apiServer.URL)
	result, err := client.GenerateVideo(context.Background(), "https://example.com/img.jpg", "test", providers.ImageToVideoOptions{})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "video/mp4", result.MimeType)
	assert.Equal(t, int32(2), callCount.Load(), "should poll exactly twice")
}
