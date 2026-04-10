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

func newI2VAcceptanceClient(t *testing.T) (*ImageToVideoClient, string) {
	t.Helper()

	apiKey := os.Getenv("XAI_API_KEY")
	if apiKey == "" {
		t.Fatal("XAI_API_KEY must be set for acceptance tests")
	}

	outputDir := filepath.Join(t.TempDir(), "grok_i2v_acceptance")
	httpClient := common.NewHTTPClient(120*time.Second, 2, 3*time.Second)
	client := NewImageToVideoClient(apiKey, httpClient, outputDir, "")

	return client, outputDir
}

// TestAcceptance_ImageToVideo_GenerateVideo animates a source image into a video.
// The source image becomes the first frame.
// NOTE: xAI generation typically takes 1–3 minutes; test timeout is set to 15 min.
func TestAcceptance_ImageToVideo_GenerateVideo(t *testing.T) {
	client, outputDir := newI2VAcceptanceClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	// Using a public sample image URL
	result, err := client.GenerateVideo(ctx,
		"https://picsum.photos/id/10/1200/800.jpg",
		"Gentle camera zoom into the scene with soft lighting",
		providers.ImageToVideoOptions{
			Duration:    "5",
			AspectRatio: DefaultAspectRatio,
		},
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	assertVideoFile(t, result.LocalPath, outputDir)
	assert.NotEmpty(t, result.ProviderURL)
	assert.Equal(t, "video/mp4", result.MimeType)
}

// TestAcceptance_ImageToVideo_PortraitMode verifies portrait aspect ratio and HD resolution.
func TestAcceptance_ImageToVideo_PortraitMode(t *testing.T) {
	client, outputDir := newI2VAcceptanceClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	result, err := client.GenerateVideo(ctx,
		"https://picsum.photos/id/10/1200/800.jpg",
		"Slow vertical pan with cinematic depth of field",
		providers.ImageToVideoOptions{
			Duration:    "5",
			AspectRatio: "9:16",
			Mode:        "720p",
		},
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	assertVideoFile(t, result.LocalPath, outputDir)
	assert.NotEmpty(t, result.ProviderURL)
}
