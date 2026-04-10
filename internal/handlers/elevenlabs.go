package handlers

import (
	"net/http"

	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/providers/elevenlabs"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/service"
	"github.com/gin-gonic/gin"
)

// SoundHandler handles ElevenLabs API requests
type SoundHandler struct {
	service *service.SoundService
}

// NewSoundHandler creates a new sound handler
func NewSoundHandler(svc *service.SoundService) *SoundHandler {
	return &SoundHandler{service: svc}
}

// GenerateSpeechRequest represents the HTTP request for TTS
type GenerateSpeechRequest struct {
	Text         string                    `json:"text" binding:"required"`
	VoiceID      string                    `json:"voice_id" binding:"required"`
	ModelID      string                    `json:"model_id,omitempty"`
	Settings     *elevenlabs.VoiceSettings `json:"voice_settings,omitempty"`
	PreviousText string                    `json:"previous_text,omitempty"`
	NextText     string                    `json:"next_text,omitempty"`
}

// GenerateSpeech handles text-to-speech generation
func (h *SoundHandler) GenerateSpeech(c *gin.Context) {
	var req GenerateSpeechRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	jobID, err := h.service.GenerateSpeech(c.Request.Context(), &elevenlabs.TTSRequest{
		Text:         req.Text,
		VoiceID:      req.VoiceID,
		ModelID:      req.ModelID,
		Settings:     req.Settings,
		PreviousText: req.PreviousText,
		NextText:     req.NextText,
	})
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "Failed to create job: "+err.Error())
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"success": true,
		"job_id":  jobID,
		"message": "Speech generation job created",
	})
}

// GenerateSoundEffectsRequest represents the HTTP request for sound effect generation
type GenerateSoundEffectsRequest struct {
	Text            string  `json:"text" binding:"required"`
	DurationSeconds float64 `json:"duration_seconds,omitempty"`
	PromptInfluence float64 `json:"prompt_influence,omitempty"`
}

// GenerateSoundEffects handles sound effect generation
func (h *SoundHandler) GenerateSoundEffects(c *gin.Context) {
	var req GenerateSoundEffectsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	jobID, err := h.service.GenerateSoundEffects(c.Request.Context(), &elevenlabs.SoundEffectRequest{
		Text:            req.Text,
		DurationSeconds: req.DurationSeconds,
		PromptInfluence: req.PromptInfluence,
	})
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "Failed to create job: "+err.Error())
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"success": true,
		"job_id":  jobID,
		"message": "Sound effect generation job created",
	})
}

// GenerateMusicRequest represents the HTTP request for music generation
type GenerateMusicRequest struct {
	Text            string  `json:"text" binding:"required"`
	DurationSeconds float64 `json:"duration_seconds,omitempty"`
}

// GenerateMusic handles music generation
func (h *SoundHandler) GenerateMusic(c *gin.Context) {
	var req GenerateMusicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	jobID, err := h.service.GenerateMusic(c.Request.Context(), &elevenlabs.MusicRequest{
		Text:            req.Text,
		DurationSeconds: req.DurationSeconds,
	})
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "Failed to create job: "+err.Error())
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"success": true,
		"job_id":  jobID,
		"message": "Music generation job created",
	})
}

// GetVoices retrieves available voices.
// Optional query param: language (e.g. "en", "es") to filter by voice language.
func (h *SoundHandler) GetVoices(c *gin.Context) {
	ctx := c.Request.Context()

	var opts *elevenlabs.GetVoicesOptions
	if lang := c.Query("language"); lang != "" {
		opts = &elevenlabs.GetVoicesOptions{Language: lang}
	}

	voices, err := h.service.GetVoices(ctx, opts)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "Failed to retrieve voices: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"voices":  voices.Voices,
	})
}
