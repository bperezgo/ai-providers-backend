package elevenlabs

// TTSRequest represents a text-to-speech request
type TTSRequest struct {
	Text         string         `json:"text" binding:"required"`
	VoiceID      string         `json:"voice_id" binding:"required"`
	ModelID      string         `json:"model_id,omitempty"`
	Settings     *VoiceSettings `json:"voice_settings,omitempty"`
	PreviousText string         `json:"previous_text,omitempty"`
	NextText     string         `json:"next_text,omitempty"`
}

// VoiceSettings represents voice configuration
type VoiceSettings struct {
	Stability       float64 `json:"stability,omitempty"`        // 0.0 - 1.0
	SimilarityBoost float64 `json:"similarity_boost,omitempty"` // 0.0 - 1.0
	Style           float64 `json:"style,omitempty"`            // 0.0 - 1.0
	UseSpeakerBoost bool    `json:"use_speaker_boost,omitempty"`
}

// TTSResponse represents the response from ElevenLabs API
type TTSResponse struct {
	AudioURL string `json:"audio_url"`
	Duration int    `json:"duration,omitempty"` // in seconds
}

// Voice represents an available voice
type Voice struct {
	VoiceID  string            `json:"voice_id"`
	Name     string            `json:"name"`
	Category string            `json:"category"`
	Labels   map[string]string `json:"labels,omitempty"`
}

// VoicesResponse represents the response from the voices endpoint
type VoicesResponse struct {
	Voices []Voice `json:"voices"`
}

// GetVoicesOptions holds optional filters for GetVoices
type GetVoicesOptions struct {
	Language string // e.g. "en", "es", "pt" — filters by labels["language"]
}

// SoundEffectRequest represents a sound effect generation request
type SoundEffectRequest struct {
	Text            string  `json:"text"`
	DurationSeconds float64 `json:"duration_seconds,omitempty"` // 0.5–22.0 seconds
	PromptInfluence float64 `json:"prompt_influence,omitempty"` // 0.0–1.0
}

// SoundEffectResponse represents the response from the sound effects API
type SoundEffectResponse struct {
	AudioURL string  `json:"audio_url"`
	Duration float64 `json:"duration,omitempty"`
}

// MusicRequest represents a music generation request
type MusicRequest struct {
	Text            string  `json:"text"`
	DurationSeconds float64 `json:"duration_seconds,omitempty"`
}

// MusicResponse represents the response from the music generation API
type MusicResponse struct {
	AudioURL string  `json:"audio_url"`
	Duration float64 `json:"duration,omitempty"`
}

// DefaultModelID is the default TTS model
const DefaultModelID = "eleven_monolingual_v1"

// DefaultVoiceSettings returns default voice settings
func DefaultVoiceSettings() *VoiceSettings {
	return &VoiceSettings{
		Stability:       0.5,
		SimilarityBoost: 0.75,
		Style:           0.0,
		UseSpeakerBoost: true,
	}
}
