package elevenlabs

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bryanperez/laguna-escondida-marketing/backend/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestHTTPClient() *common.HTTPClient {
	return common.NewHTTPClient(10*time.Second, 0, 0)
}

func TestGenerateSpeech_Success(t *testing.T) {
	audioContent := []byte("fake-mp3-audio-data")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "test-api-key", r.Header.Get("xi-api-key"))
		assert.Equal(t, "audio/mpeg", r.Header.Get("Accept"))
		assert.Contains(t, r.URL.Path, "/text-to-speech/voice123")

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var req map[string]any
		require.NoError(t, json.Unmarshal(body, &req))
		assert.Equal(t, "Hello world", req["text"])
		assert.Equal(t, DefaultModelID, req["model_id"])
		assert.NotNil(t, req["voice_settings"])

		w.Header().Set("Content-Type", "audio/mpeg")
		w.WriteHeader(http.StatusOK)
		w.Write(audioContent)
	}))
	defer server.Close()

	outputDir := t.TempDir()
	client := NewClient("test-api-key", newTestHTTPClient(), outputDir, server.URL)

	resp, err := client.GenerateSpeech(context.Background(), &TTSRequest{
		Text:    "Hello world",
		VoiceID: "voice123",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.AudioURL)

	// Verify the audio file exists and has correct content
	data, err := os.ReadFile(resp.AudioURL)
	require.NoError(t, err)
	assert.Equal(t, audioContent, data)
	assert.True(t, filepath.IsAbs(resp.AudioURL))
	assert.Contains(t, resp.AudioURL, outputDir)
}

func TestGenerateSpeech_CustomModelAndSettings(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var req map[string]any
		require.NoError(t, json.Unmarshal(body, &req))
		assert.Equal(t, "eleven_multilingual_v2", req["model_id"])
		assert.NotNil(t, req["voice_settings"])

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("audio"))
	}))
	defer server.Close()

	client := NewClient("key", newTestHTTPClient(), t.TempDir(), server.URL)

	resp, err := client.GenerateSpeech(context.Background(), &TTSRequest{
		Text:    "Hola mundo",
		VoiceID: "voice456",
		ModelID: "eleven_multilingual_v2",
		Settings: &VoiceSettings{
			Stability:       0.8,
			SimilarityBoost: 0.9,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)

	_, err = os.Stat(resp.AudioURL)
	assert.NoError(t, err)
}

func TestGenerateSpeech_WithPreviousAndNextText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var req map[string]any
		require.NoError(t, json.Unmarshal(body, &req))
		assert.Equal(t, "Hello world", req["text"])
		assert.Equal(t, "Before text", req["previous_text"])
		assert.Equal(t, "After text", req["next_text"])

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("audio"))
	}))
	defer server.Close()

	client := NewClient("key", newTestHTTPClient(), t.TempDir(), server.URL)

	resp, err := client.GenerateSpeech(context.Background(), &TTSRequest{
		Text:         "Hello world",
		VoiceID:      "voice123",
		PreviousText: "Before text",
		NextText:     "After text",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.AudioURL)
}

func TestGenerateSpeech_OmitsPreviousNextTextWhenEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var req map[string]any
		require.NoError(t, json.Unmarshal(body, &req))
		_, hasPrevious := req["previous_text"]
		assert.False(t, hasPrevious, "previous_text should not be sent when empty")
		_, hasNext := req["next_text"]
		assert.False(t, hasNext, "next_text should not be sent when empty")

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("audio"))
	}))
	defer server.Close()

	client := NewClient("key", newTestHTTPClient(), t.TempDir(), server.URL)

	resp, err := client.GenerateSpeech(context.Background(), &TTSRequest{
		Text:    "Hello world",
		VoiceID: "voice123",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestGenerateSpeech_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"detail":{"message":"Invalid API key"}}`))
	}))
	defer server.Close()

	client := NewClient("bad-key", newTestHTTPClient(), t.TempDir(), server.URL)

	resp, err := client.GenerateSpeech(context.Background(), &TTSRequest{
		Text:    "Hello",
		VoiceID: "voice123",
	})

	require.Error(t, err)
	assert.Nil(t, resp)

	var apiErr *common.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusUnauthorized, apiErr.StatusCode)
	assert.Equal(t, "elevenlabs", apiErr.Provider)
}

func TestGenerateSoundEffect_Success(t *testing.T) {
	audioContent := []byte("fake-sfx-audio")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "test-key", r.Header.Get("xi-api-key"))
		assert.Contains(t, r.URL.Path, "/sound-generation")

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var req map[string]any
		require.NoError(t, json.Unmarshal(body, &req))
		assert.Equal(t, "Thunder rolling in the distance", req["text"])
		assert.Equal(t, 5.0, req["duration_seconds"])
		assert.Equal(t, 0.7, req["prompt_influence"])

		w.WriteHeader(http.StatusOK)
		w.Write(audioContent)
	}))
	defer server.Close()

	outputDir := t.TempDir()
	client := NewClient("test-key", newTestHTTPClient(), outputDir, server.URL)

	resp, err := client.GenerateSoundEffect(context.Background(), &SoundEffectRequest{
		Text:            "Thunder rolling in the distance",
		DurationSeconds: 5.0,
		PromptInfluence: 0.7,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)

	data, err := os.ReadFile(resp.AudioURL)
	require.NoError(t, err)
	assert.Equal(t, audioContent, data)
	assert.Contains(t, resp.AudioURL, outputDir)
}

func TestGenerateSoundEffect_WithoutOptionalFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var req map[string]any
		require.NoError(t, json.Unmarshal(body, &req))
		assert.Equal(t, "Birds chirping", req["text"])
		_, hasDuration := req["duration_seconds"]
		assert.False(t, hasDuration, "duration_seconds should not be sent when zero")
		_, hasInfluence := req["prompt_influence"]
		assert.False(t, hasInfluence, "prompt_influence should not be sent when zero")

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("audio"))
	}))
	defer server.Close()

	client := NewClient("key", newTestHTTPClient(), t.TempDir(), server.URL)

	resp, err := client.GenerateSoundEffect(context.Background(), &SoundEffectRequest{
		Text: "Birds chirping",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)

	_, err = os.Stat(resp.AudioURL)
	assert.NoError(t, err)
}

func TestGenerateSoundEffect_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"detail":"rate limit exceeded"}`))
	}))
	defer server.Close()

	client := NewClient("key", newTestHTTPClient(), t.TempDir(), server.URL)

	resp, err := client.GenerateSoundEffect(context.Background(), &SoundEffectRequest{
		Text: "Thunder",
	})

	require.Error(t, err)
	assert.Nil(t, resp)

	var apiErr *common.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusTooManyRequests, apiErr.StatusCode)
}

func TestGenerateMusic_Success(t *testing.T) {
	audioContent := []byte("fake-music-audio")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "music-key", r.Header.Get("xi-api-key"))
		assert.Contains(t, r.URL.Path, "/music-generation")

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var req map[string]any
		require.NoError(t, json.Unmarshal(body, &req))
		assert.Equal(t, "Upbeat tropical music", req["prompt"])
		assert.Equal(t, float64(30000), req["music_length_ms"])

		w.WriteHeader(http.StatusOK)
		w.Write(audioContent)
	}))
	defer server.Close()

	outputDir := t.TempDir()
	client := NewClient("music-key", newTestHTTPClient(), outputDir, server.URL)

	resp, err := client.GenerateMusic(context.Background(), &MusicRequest{
		Text:            "Upbeat tropical music",
		DurationSeconds: 30,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)

	data, err := os.ReadFile(resp.AudioURL)
	require.NoError(t, err)
	assert.Equal(t, audioContent, data)
	assert.Contains(t, resp.AudioURL, outputDir)
}

func TestGenerateMusic_WithoutDuration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var req map[string]any
		require.NoError(t, json.Unmarshal(body, &req))
		assert.Equal(t, "Calm ambient", req["prompt"])
		_, hasLength := req["music_length_ms"]
		assert.False(t, hasLength, "music_length_ms should not be sent when duration is zero")

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("audio"))
	}))
	defer server.Close()

	client := NewClient("key", newTestHTTPClient(), t.TempDir(), server.URL)

	resp, err := client.GenerateMusic(context.Background(), &MusicRequest{
		Text: "Calm ambient",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)

	_, err = os.Stat(resp.AudioURL)
	assert.NoError(t, err)
}

func TestGenerateMusic_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"detail":"internal server error"}`))
	}))
	defer server.Close()

	client := NewClient("key", newTestHTTPClient(), t.TempDir(), server.URL)

	resp, err := client.GenerateMusic(context.Background(), &MusicRequest{
		Text: "Jazz",
	})

	require.Error(t, err)
	assert.Nil(t, resp)

	var apiErr *common.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusInternalServerError, apiErr.StatusCode)
}

func TestGetVoices_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "voices-key", r.Header.Get("xi-api-key"))
		assert.Equal(t, "/voices", r.URL.Path)

		resp := VoicesResponse{
			Voices: []Voice{
				{VoiceID: "v1", Name: "Rachel", Labels: map[string]string{"language": "en"}},
				{VoiceID: "v2", Name: "María", Labels: map[string]string{"language": "es"}},
				{VoiceID: "v3", Name: "James", Labels: map[string]string{"language": "en"}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("voices-key", newTestHTTPClient(), t.TempDir(), server.URL)

	resp, err := client.GetVoices(context.Background(), nil)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Voices, 3)
}

func TestGetVoices_FilterByLanguage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := VoicesResponse{
			Voices: []Voice{
				{VoiceID: "v1", Name: "Rachel", Labels: map[string]string{"language": "en"}},
				{VoiceID: "v2", Name: "María", Labels: map[string]string{"language": "es"}},
				{VoiceID: "v3", Name: "James", Labels: map[string]string{"language": "en"}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("key", newTestHTTPClient(), t.TempDir(), server.URL)

	resp, err := client.GetVoices(context.Background(), &GetVoicesOptions{Language: "es"})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Voices, 1)
	assert.Equal(t, "María", resp.Voices[0].Name)
	assert.Equal(t, "v2", resp.Voices[0].VoiceID)
}

func TestGetVoices_FilterReturnsEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := VoicesResponse{
			Voices: []Voice{
				{VoiceID: "v1", Name: "Rachel", Labels: map[string]string{"language": "en"}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("key", newTestHTTPClient(), t.TempDir(), server.URL)

	resp, err := client.GetVoices(context.Background(), &GetVoicesOptions{Language: "fr"})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Empty(t, resp.Voices)
}

func TestGetVoices_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"detail":"forbidden"}`))
	}))
	defer server.Close()

	client := NewClient("key", newTestHTTPClient(), t.TempDir(), server.URL)

	resp, err := client.GetVoices(context.Background(), nil)

	require.Error(t, err)
	assert.Nil(t, resp)

	var apiErr *common.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusForbidden, apiErr.StatusCode)
}

func TestNewClient_DefaultBaseURL(t *testing.T) {
	client := NewClient("key", newTestHTTPClient(), "/tmp/out", "")
	assert.Equal(t, defaultBaseURL, client.baseURL)
}

func TestNewClient_CustomBaseURL(t *testing.T) {
	client := NewClient("key", newTestHTTPClient(), "/tmp/out", "https://custom.api.com")
	assert.Equal(t, "https://custom.api.com", client.baseURL)
}

func TestProviderName(t *testing.T) {
	client := NewClient("key", newTestHTTPClient(), "/tmp/out", "")
	assert.Equal(t, "elevenlabs", client.ProviderName())
}

func TestSaveAudioToFile_CreatesOutputDir(t *testing.T) {
	outputDir := filepath.Join(t.TempDir(), "nested", "audio", "output")
	audioContent := []byte("test-audio-content")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(audioContent)
	}))
	defer server.Close()

	client := NewClient("key", newTestHTTPClient(), outputDir, server.URL)

	resp, err := client.GenerateSpeech(context.Background(), &TTSRequest{
		Text:    "Test",
		VoiceID: "v1",
	})

	require.NoError(t, err)

	// Verify nested directory was created
	info, err := os.Stat(outputDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	// Verify file is inside outputDir
	assert.Contains(t, resp.AudioURL, outputDir)

	data, err := os.ReadFile(resp.AudioURL)
	require.NoError(t, err)
	assert.Equal(t, audioContent, data)
}

func TestAudioFileNaming(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("audio"))
	}))
	defer server.Close()

	client := NewClient("key", newTestHTTPClient(), t.TempDir(), server.URL)

	resp, err := client.GenerateSpeech(context.Background(), &TTSRequest{
		Text:    "Test",
		VoiceID: "v1",
	})

	require.NoError(t, err)

	filename := filepath.Base(resp.AudioURL)
	assert.Contains(t, filename, "elevenlabs_")
	assert.Contains(t, filename, ".mp3")
}
