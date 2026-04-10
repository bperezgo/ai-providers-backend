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

func newRefAcceptanceClient(t *testing.T) (*ReferenceVideoClient, string) {
	t.Helper()

	apiKey := os.Getenv("XAI_API_KEY")
	if apiKey == "" {
		t.Fatal("XAI_API_KEY must be set for acceptance tests")
	}

	outputDir := filepath.Join(t.TempDir(), "grok_ref_acceptance")
	httpClient := common.NewHTTPClient(120*time.Second, 2, 3*time.Second)
	client := NewReferenceVideoClient(apiKey, httpClient, outputDir, "")

	return client, outputDir
}

// TestAcceptance_ReferenceVideo_SingleReference generates a video influenced by a single
// reference image. The reference affects visual style without locking the first frame.
// NOTE: xAI generation typically takes 1–3 minutes; test timeout is set to 15 min.
func TestAcceptance_ReferenceVideo_SingleReference(t *testing.T) {
	client, outputDir := newRefAcceptanceClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	result, err := client.GenerateVideoFromReferences(ctx,
		"A serene landscape in the style of <IMAGE_1> with soft morning light",
		[]string{
			"https://picsum.photos/id/10/1200/800.jpg",
		},
		providers.ReferenceImageVideoOptions{
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

// TestAcceptance_ReferenceVideo_MultipleReferences generates a video influenced by
// two reference images using <IMAGE_1> and <IMAGE_2> placeholders.
func TestAcceptance_ReferenceVideo_MultipleReferences(t *testing.T) {
	client, outputDir := newRefAcceptanceClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	result, err := client.GenerateVideoFromReferences(ctx,
		"A scene combining elements from <IMAGE_1> and <IMAGE_2> with cinematic movement",
		[]string{
			"https://picsum.photos/id/10/1200/800.jpg",
			"https://picsum.photos/id/20/1200/800.jpg",
		},
		providers.ReferenceImageVideoOptions{
			Duration:    "5",
			AspectRatio: "16:9",
			Mode:        "720p",
		},
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	assertVideoFile(t, result.LocalPath, outputDir)
	assert.NotEmpty(t, result.ProviderURL)
}
