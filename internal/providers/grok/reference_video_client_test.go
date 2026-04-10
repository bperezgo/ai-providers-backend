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

func newRefTestHTTPClient() *common.HTTPClient {
	return common.NewHTTPClient(10*time.Second, 0, 0)
}

// --- Constructor & basic ---

func TestNewReferenceVideoClient_DefaultBaseURL(t *testing.T) {
	client := NewReferenceVideoClient("key", newRefTestHTTPClient(), t.TempDir(), "")
	assert.Equal(t, DefaultXAIBaseURL, client.baseURL)
}

func TestNewReferenceVideoClient_CustomBaseURL(t *testing.T) {
	client := NewReferenceVideoClient("key", newRefTestHTTPClient(), t.TempDir(), "https://custom.api")
	assert.Equal(t, "https://custom.api", client.baseURL)
}

func TestReferenceVideoClient_ProviderName(t *testing.T) {
	client := NewReferenceVideoClient("key", newRefTestHTTPClient(), t.TempDir(), "")
	assert.Equal(t, "grok-ref", client.ProviderName())
}

// --- submit ---

func TestReferenceVideoClient_Submit_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "/videos/generations", r.URL.Path)

		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, ModelID, body["model"])
		assert.Equal(t, "A cat <IMAGE_1> playing with <IMAGE_2>", body["prompt"])
		assert.Equal(t, float64(DefaultDuration), body["duration"])
		assert.Equal(t, DefaultAspectRatio, body["aspect_ratio"])
		assert.Equal(t, DefaultResolution, body["resolution"])

		// Verify reference_images structure
		refImages, ok := body["reference_images"].([]any)
		require.True(t, ok)
		require.Len(t, refImages, 2)

		img1 := refImages[0].(map[string]any)
		assert.Equal(t, "https://example.com/cat.jpg", img1["url"])
		img2 := refImages[1].(map[string]any)
		assert.Equal(t, "https://example.com/toy.jpg", img2["url"])

		// Must NOT contain image or source_image
		_, hasImage := body["image"]
		assert.False(t, hasImage)
		_, hasSourceImage := body["source_image"]
		assert.False(t, hasSourceImage)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(SubmitResponse{RequestID: "ref-123"})
	}))
	defer server.Close()

	client := NewReferenceVideoClient("test-key", newRefTestHTTPClient(), t.TempDir(), server.URL)
	resp, err := client.submit(context.Background(), &ReferenceVideoGenerateRequest{
		Prompt: "A cat <IMAGE_1> playing with <IMAGE_2>",
		ReferenceImages: []ReferenceImage{
			{URL: "https://example.com/cat.jpg"},
			{URL: "https://example.com/toy.jpg"},
		},
		Duration:    DefaultDuration,
		AspectRatio: DefaultAspectRatio,
		Resolution:  DefaultResolution,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "ref-123", resp.RequestID)
}

func TestReferenceVideoClient_Submit_SingleReference(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))

		refImages, ok := body["reference_images"].([]any)
		require.True(t, ok)
		require.Len(t, refImages, 1)

		json.NewEncoder(w).Encode(SubmitResponse{RequestID: "ref-single"})
	}))
	defer server.Close()

	client := NewReferenceVideoClient("test-key", newRefTestHTTPClient(), t.TempDir(), server.URL)
	resp, err := client.submit(context.Background(), &ReferenceVideoGenerateRequest{
		Prompt: "Style like <IMAGE_1>",
		ReferenceImages: []ReferenceImage{
			{URL: "https://example.com/style.jpg"},
		},
		Duration:    DefaultDuration,
		AspectRatio: DefaultAspectRatio,
		Resolution:  DefaultResolution,
	})

	require.NoError(t, err)
	assert.Equal(t, "ref-single", resp.RequestID)
}

func TestReferenceVideoClient_Submit_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"cannot combine image and reference_images"}`))
	}))
	defer server.Close()

	client := NewReferenceVideoClient("test-key", newRefTestHTTPClient(), t.TempDir(), server.URL)
	_, err := client.submit(context.Background(), &ReferenceVideoGenerateRequest{
		Prompt: "test",
		ReferenceImages: []ReferenceImage{
			{URL: "https://example.com/img.jpg"},
		},
		Duration: DefaultDuration,
	})

	require.Error(t, err)
	var apiErr *common.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
	assert.Equal(t, "grok-ref", apiErr.Provider)
}

// --- GenerateVideoFromReferences ---

func TestReferenceVideoClient_GenerateVideo_RequiresReferenceImages(t *testing.T) {
	client := NewReferenceVideoClient("key", newRefTestHTTPClient(), t.TempDir(), "")
	_, err := client.GenerateVideoFromReferences(context.Background(), "prompt", nil, providers.ReferenceImageVideoOptions{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one reference image URL is required")
}

func TestReferenceVideoClient_GenerateVideo_EmptySlice(t *testing.T) {
	client := NewReferenceVideoClient("key", newRefTestHTTPClient(), t.TempDir(), "")
	_, err := client.GenerateVideoFromReferences(context.Background(), "prompt", []string{}, providers.ReferenceImageVideoOptions{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one reference image URL is required")
}

func TestReferenceVideoClient_GenerateVideo_Success(t *testing.T) {
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
			assert.Equal(t, "A scene like <IMAGE_1> and <IMAGE_2>", body["prompt"])

			refImages, ok := body["reference_images"].([]any)
			require.True(t, ok)
			require.Len(t, refImages, 2)

			json.NewEncoder(w).Encode(SubmitResponse{RequestID: "ref-abc"})
			return
		}
		json.NewEncoder(w).Encode(GenerateResponse{
			Status: StatusDone,
			Video:  &GrokVideo{URL: videoServer.URL + "/video.mp4", Duration: 5},
		})
	}))
	defer apiServer.Close()

	outputDir := t.TempDir()
	client := NewReferenceVideoClient("test-key", newRefTestHTTPClient(), outputDir, apiServer.URL)

	result, err := client.GenerateVideoFromReferences(
		context.Background(),
		"A scene like <IMAGE_1> and <IMAGE_2>",
		[]string{"https://example.com/ref1.jpg", "https://example.com/ref2.jpg"},
		providers.ReferenceImageVideoOptions{
			Duration:    "5",
			AspectRatio: "16:9",
			Mode:        "480p",
		},
	)

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

func TestReferenceVideoClient_GenerateVideo_DefaultOptions(t *testing.T) {
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

		json.NewEncoder(w).Encode(SubmitResponse{RequestID: "ref-def"})
	}))
	defer server.Close()

	client := NewReferenceVideoClient("test-key", newRefTestHTTPClient(), t.TempDir(), server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, _ = client.GenerateVideoFromReferences(ctx, "test <IMAGE_1>", []string{"https://example.com/img.jpg"}, providers.ReferenceImageVideoOptions{})
}

func TestReferenceVideoClient_GenerateVideo_SubmitAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"quota_exceeded"}`))
	}))
	defer server.Close()

	client := NewReferenceVideoClient("test-key", newRefTestHTTPClient(), t.TempDir(), server.URL)
	result, err := client.GenerateVideoFromReferences(
		context.Background(),
		"test <IMAGE_1>",
		[]string{"https://example.com/img.jpg"},
		providers.ReferenceImageVideoOptions{},
	)

	require.Error(t, err)
	assert.Nil(t, result)
	var apiErr *common.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusForbidden, apiErr.StatusCode)
}

func TestReferenceVideoClient_GenerateVideo_PollPendingThenDone(t *testing.T) {
	fakeMP4 := []byte{0x00, 0x00, 0x00, 0x18, 0x66, 0x74, 0x79, 0x70}
	var callCount atomic.Int32

	videoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(fakeMP4)
	}))
	defer videoServer.Close()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			json.NewEncoder(w).Encode(SubmitResponse{RequestID: "ref-poll"})
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

	client := NewReferenceVideoClient("test-key", newRefTestHTTPClient(), t.TempDir(), apiServer.URL)
	result, err := client.GenerateVideoFromReferences(
		context.Background(),
		"test <IMAGE_1>",
		[]string{"https://example.com/img.jpg"},
		providers.ReferenceImageVideoOptions{},
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "video/mp4", result.MimeType)
	assert.Equal(t, int32(2), callCount.Load(), "should poll exactly twice")
}
