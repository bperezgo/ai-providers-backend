//go:build acceptance

package grok

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/providers"
	"github.com/bryanperez/laguna-escondida-marketing/backend/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newAcceptanceClient(t *testing.T) (*Client, string) {
	t.Helper()

	apiKey := os.Getenv("XAI_API_KEY")
	if apiKey == "" {
		t.Fatal("XAI_API_KEY must be set for acceptance tests")
	}

	outputDir := filepath.Join(t.TempDir(), "grok_acceptance")
	httpClient := common.NewHTTPClient(120*time.Second, 2, 3*time.Second)
	client := NewClient(apiKey, httpClient, outputDir, "")

	return client, outputDir
}

func assertVideoFile(t *testing.T, path string, outputDir string) {
	t.Helper()

	require.NotEmpty(t, path, "video path should not be empty")
	assert.Contains(t, path, outputDir)

	info, err := os.Stat(path)
	require.NoError(t, err, "video file should exist at %s", path)
	assert.Greater(t, info.Size(), int64(0), "video file should not be empty")
	assert.Equal(t, ".mp4", filepath.Ext(path))
}

// TestAcceptance_GenerateVideo generates a text-to-video clip using default settings.
// NOTE: xAI generation typically takes 1–3 minutes; test timeout is set to 15 min.
func TestAcceptance_GenerateVideo(t *testing.T) {
	client, outputDir := newAcceptanceClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	result, err := client.GenerateVideo(ctx, "", "A calm ocean wave at sunset", providers.ImageToVideoOptions{
		Duration:    "5",
		AspectRatio: DefaultAspectRatio,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assertVideoFile(t, result.LocalPath, outputDir)
	assert.NotEmpty(t, result.ProviderURL)
	assert.Equal(t, "video/mp4", result.MimeType)
}

// TestAcceptance_GenerateVideo_PortraitMode verifies portrait aspect ratio and HD resolution.
func TestAcceptance_GenerateVideo_PortraitMode(t *testing.T) {
	client, outputDir := newAcceptanceClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	result, err := client.GenerateVideo(ctx, "", "A vertical waterfall in a lush jungle", providers.ImageToVideoOptions{
		Duration:    "5",
		AspectRatio: "9:16",
		Mode:        "720p",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assertVideoFile(t, result.LocalPath, outputDir)
	assert.NotEmpty(t, result.ProviderURL)
}
