package handlers

import (
	"net/http"

	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/service"
	"github.com/gin-gonic/gin"
)

// NanoBananaHandler handles Nano Banana Pro (via Fal.ai) API requests
type NanoBananaHandler struct {
	service *service.NanoBananaService
}

// NewNanoBananaHandler creates a new Nano Banana handler
func NewNanoBananaHandler(svc *service.NanoBananaService) *NanoBananaHandler {
	return &NanoBananaHandler{service: svc}
}

// NanoBananaGenerateRequest represents the HTTP request for Nano Banana image generation
type NanoBananaGenerateRequest struct {
	Prompt            string  `json:"prompt" binding:"required"`
	NegativePrompt    string  `json:"negative_prompt,omitempty"`
	ImageSize         string  `json:"image_size,omitempty"`          // "square", "square_hd", "portrait_4_3", "portrait_16_9", "landscape_4_3", "landscape_16_9"
	NumImages         int     `json:"num_images,omitempty"`          // 1-4, default 1
	NumInferenceSteps int     `json:"num_inference_steps,omitempty"` // default 28
	GuidanceScale     float64 `json:"guidance_scale,omitempty"`      // default 3.5
	Seed              int64   `json:"seed,omitempty"`
}

// GenerateImage handles text-to-image generation via Nano Banana Pro on Fal.ai
func (h *NanoBananaHandler) GenerateImage(c *gin.Context) {
	var req NanoBananaGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	jobID, err := h.service.GenerateImage(c.Request.Context(), &service.NanoBananaRequest{
		Prompt:            req.Prompt,
		NegativePrompt:    req.NegativePrompt,
		ImageSize:         req.ImageSize,
		NumImages:         req.NumImages,
		NumInferenceSteps: req.NumInferenceSteps,
		GuidanceScale:     req.GuidanceScale,
		Seed:              req.Seed,
	})
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "Failed to create job: "+err.Error())
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"success": true,
		"job_id":  jobID,
		"message": "Nano Banana image generation job created",
	})
}
