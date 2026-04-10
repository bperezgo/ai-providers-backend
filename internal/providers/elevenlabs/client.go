package elevenlabs

import (
	"bytes"
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

const (
	defaultBaseURL      = "https://api.elevenlabs.io/v1"
	ttsEndpoint         = "/text-to-speech"
	soundEffectEndpoint = "/sound-generation"
	musicEndpoint       = "/music"
)

// Client is the ElevenLabs API client
type Client struct {
	apiKey     string
	httpClient *common.HTTPClient
	outputDir  string
	baseURL    string
}

// NewClient creates a new ElevenLabs client.
// If baseURL is empty, the default ElevenLabs API URL is used.
func NewClient(apiKey string, httpClient *common.HTTPClient, outputDir string, baseURL string) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		apiKey:     apiKey,
		httpClient: httpClient,
		outputDir:  outputDir,
		baseURL:    baseURL,
	}
}

// ProviderName returns the provider identifier.
func (c *Client) ProviderName() string { return "elevenlabs" }

// GenerateSpeech generates speech from text
func (c *Client) GenerateSpeech(ctx context.Context, req *TTSRequest) (*TTSResponse, error) {
	// Apply defaults
	if req.ModelID == "" {
		req.ModelID = DefaultModelID
	}
	if req.Settings == nil {
		req.Settings = DefaultVoiceSettings()
	}

	// Prepare request body
	body := map[string]any{
		"text":     req.Text,
		"model_id": req.ModelID,
	}

	if req.Settings != nil {
		body["voice_settings"] = req.Settings
	}
	if req.PreviousText != "" {
		body["previous_text"] = req.PreviousText
	}
	if req.NextText != "" {
		body["next_text"] = req.NextText
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Build URL
	url := fmt.Sprintf("%s%s/%s", c.baseURL, ttsEndpoint, req.VoiceID)

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("xi-api-key", c.apiKey)
	httpReq.Header.Set("Accept", "audio/mpeg")

	// Execute request
	resp, err := c.httpClient.DoWithRetry(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, &common.APIError{
			Provider:   "elevenlabs",
			StatusCode: resp.StatusCode,
			Message:    string(bodyBytes),
		}
	}

	// Save audio to file
	audioPath, err := c.saveAudioToFile(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to save audio: %w", err)
	}

	return &TTSResponse{
		AudioURL: audioPath,
	}, nil
}

// saveAudioToFile saves the audio response to a file
func (c *Client) saveAudioToFile(reader io.Reader) (string, error) {
	// Ensure output directory exists
	if err := os.MkdirAll(c.outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate filename
	filename := fmt.Sprintf("elevenlabs_%s_%d.mp3", uuid.Must(uuid.NewV7()).String(), time.Now().Unix())
	filepath := filepath.Join(c.outputDir, filename)

	// Create file
	file, err := os.Create(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Copy audio data to file
	if _, err := io.Copy(file, reader); err != nil {
		return "", fmt.Errorf("failed to write audio data: %w", err)
	}

	return filepath, nil
}

// GenerateSoundEffect generates a sound effect from a text description
func (c *Client) GenerateSoundEffect(ctx context.Context, req *SoundEffectRequest) (*SoundEffectResponse, error) {
	body := map[string]any{
		"text": req.Text,
	}
	if req.DurationSeconds != 0 {
		body["duration_seconds"] = req.DurationSeconds
	}
	if req.PromptInfluence != 0 {
		body["prompt_influence"] = req.PromptInfluence
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s%s", c.baseURL, soundEffectEndpoint)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("xi-api-key", c.apiKey)
	httpReq.Header.Set("Accept", "audio/mpeg")

	resp, err := c.httpClient.DoWithRetry(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, &common.APIError{
			Provider:   "elevenlabs",
			StatusCode: resp.StatusCode,
			Message:    string(bodyBytes),
		}
	}

	audioPath, err := c.saveAudioToFile(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to save audio: %w", err)
	}

	return &SoundEffectResponse{
		AudioURL: audioPath,
	}, nil
}

// GenerateMusic generates music from a text prompt
func (c *Client) GenerateMusic(ctx context.Context, req *MusicRequest) (*MusicResponse, error) {
	body := map[string]any{
		"prompt": req.Text,
	}
	if req.DurationSeconds != 0 {
		body["music_length_ms"] = int(req.DurationSeconds * 1000)
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s%s", c.baseURL, musicEndpoint)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("xi-api-key", c.apiKey)
	httpReq.Header.Set("Accept", "audio/mpeg")

	resp, err := c.httpClient.DoWithRetry(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, &common.APIError{
			Provider:   "elevenlabs",
			StatusCode: resp.StatusCode,
			Message:    string(bodyBytes),
		}
	}

	audioPath, err := c.saveAudioToFile(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to save audio: %w", err)
	}

	return &MusicResponse{
		AudioURL: audioPath,
	}, nil
}

// GetVoices retrieves available voices (utility method).
// Pass opts with a Language code (e.g. "en", "es") to filter client-side by labels["language"].
func (c *Client) GetVoices(ctx context.Context, opts *GetVoicesOptions) (*VoicesResponse, error) {
	url := fmt.Sprintf("%s/voices", c.baseURL)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("xi-api-key", c.apiKey)

	resp, err := c.httpClient.DoWithRetry(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, &common.APIError{
			Provider:   "elevenlabs",
			StatusCode: resp.StatusCode,
			Message:    string(bodyBytes),
		}
	}

	var voicesResp VoicesResponse
	if err := json.NewDecoder(resp.Body).Decode(&voicesResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if opts != nil && opts.Language != "" {
		filtered := voicesResp.Voices[:0]
		for _, v := range voicesResp.Voices {
			if v.Labels["language"] == opts.Language {
				filtered = append(filtered, v)
			}
		}
		voicesResp.Voices = filtered
	}

	return &voicesResp, nil
}
