//go:build acceptance

package elevenlabs

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bryanperez/laguna-escondida-marketing/backend/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// Rachel — default ElevenLabs English voice, always available.
	acceptanceVoiceID = "21m00Tcm4TlvDq8ikWAM"
)

func newAcceptanceClient(t *testing.T) (*Client, string) {
	t.Helper()

	apiKey := os.Getenv("ELEVENLABS_API_KEY")
	if apiKey == "" {
		t.Fatal("ELEVENLABS_API_KEY must be set for acceptance tests")
	}

	outputDir := filepath.Join(os.TempDir(), "elevenlabs_acceptance")
	httpClient := common.NewHTTPClient(60*time.Second, 2, 3*time.Second)
	client := NewClient(apiKey, httpClient, outputDir, "")

	return client, outputDir
}

func assertAudioFile(t *testing.T, path string, outputDir string) {
	t.Helper()

	require.NotEmpty(t, path, "audio path should not be empty")
	assert.Contains(t, path, outputDir)

	info, err := os.Stat(path)
	require.NoError(t, err, "audio file should exist at %s", path)
	assert.Greater(t, info.Size(), int64(0), "audio file should not be empty")
	assert.Equal(t, ".mp3", filepath.Ext(path))
}

func TestAcceptance_GenerateSpeech(t *testing.T) {
	client, outputDir := newAcceptanceClient(t)
	ctx := context.Background()

	resp, err := client.GenerateSpeech(ctx, &TTSRequest{
		Text:    "Hello, this is an acceptance test.",
		VoiceID: acceptanceVoiceID,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assertAudioFile(t, resp.AudioURL, outputDir)
}

func TestAcceptance_GenerateSpeech_CustomModel(t *testing.T) {
	client, outputDir := newAcceptanceClient(t)
	ctx := context.Background()

	resp, err := client.GenerateSpeech(ctx, &TTSRequest{
		Text:    "Hola, esto es una prueba.",
		VoiceID: acceptanceVoiceID,
		ModelID: "eleven_multilingual_v2",
		Settings: &VoiceSettings{
			Stability:       0.6,
			SimilarityBoost: 0.8,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assertAudioFile(t, resp.AudioURL, outputDir)
}

func TestAcceptance_GenerateSpeech_WithPreviousAndNextText(t *testing.T) {
	client, outputDir := newAcceptanceClient(t)
	ctx := context.Background()

	resp, err := client.GenerateSpeech(ctx, &TTSRequest{
		Text:         "Treinta años rodeada de las personas que más quiero.",
		VoiceID:      acceptanceVoiceID,
		ModelID:      "eleven_multilingual_v2",
		PreviousText: "Me dijeron que era solo una cena. Mis amigas habían preparado todo esto.",
		NextText:     "La comida, el lugar, todo fue perfecto.",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assertAudioFile(t, resp.AudioURL, outputDir)
}

func TestAcceptance_GenerateSoundEffect(t *testing.T) {
	client, outputDir := newAcceptanceClient(t)
	ctx := context.Background()

	resp, err := client.GenerateSoundEffect(ctx, &SoundEffectRequest{
		Text:            "A short door knock",
		DurationSeconds: 1.0,
		PromptInfluence: 0.5,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assertAudioFile(t, resp.AudioURL, outputDir)
}

func TestAcceptance_GenerateMusic(t *testing.T) {
	client, outputDir := newAcceptanceClient(t)
	ctx := context.Background()

	resp, err := client.GenerateMusic(ctx, &MusicRequest{
		Text:            "Short calm ambient loop",
		DurationSeconds: 5,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assertAudioFile(t, resp.AudioURL, outputDir)
}

func TestAcceptance_GetVoices(t *testing.T) {
	client, _ := newAcceptanceClient(t)
	ctx := context.Background()

	resp, err := client.GetVoices(ctx, nil)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.Voices, "should return at least one voice")

	// Verify voice structure
	for _, v := range resp.Voices {
		assert.NotEmpty(t, v.VoiceID)
		assert.NotEmpty(t, v.Name)
	}
}

func TestAcceptance_GetVoices_FilterByLanguage(t *testing.T) {
	client, _ := newAcceptanceClient(t)
	ctx := context.Background()

	resp, err := client.GetVoices(ctx, &GetVoicesOptions{Language: "en"})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.Voices, "should return English voices")

	for _, v := range resp.Voices {
		assert.Equal(t, "en", v.Labels["language"], "voice %s should be English", v.Name)
	}
}
