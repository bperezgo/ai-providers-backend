package handlers

import (
	"net/http"

	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/service"
	"github.com/gin-gonic/gin"
)

// GrokHandler handles xAI Grok video generation requests.
type GrokHandler struct {
	service *service.GrokService
}

// NewGrokHandler creates a new Grok handler.
func NewGrokHandler(svc *service.GrokService) *GrokHandler {
	return &GrokHandler{service: svc}
}

// GrokTextToVideoRequest represents the HTTP request for text-to-video generation.
type GrokTextToVideoRequest struct {
	Prompt      string `json:"prompt" binding:"required"`
	SourceImage string `json:"source_image,omitempty"`
	Duration    string `json:"duration,omitempty"`
	AspectRatio string `json:"aspect_ratio,omitempty"`
	Resolution  string `json:"resolution,omitempty"`
}

// GenerateTextToVideo handles text-to-video generation via xAI Grok.
func (h *GrokHandler) GenerateTextToVideo(c *gin.Context) {
	var req GrokTextToVideoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	jobID, err := h.service.GenerateTextToVideo(c.Request.Context(), &service.GrokTextToVideoRequest{
		Prompt:      req.Prompt,
		SourceImage: req.SourceImage,
		Duration:    req.Duration,
		AspectRatio: req.AspectRatio,
		Resolution:  req.Resolution,
	})
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "Failed to create job: "+err.Error())
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"success": true,
		"job_id":  jobID,
		"message": "Grok text-to-video job created",
	})
}

// GrokImageToVideoRequest represents the HTTP request for image-to-video generation.
type GrokImageToVideoRequest struct {
	ImageURL    string `json:"image_url" binding:"required"`
	Prompt      string `json:"prompt,omitempty"`
	Duration    string `json:"duration,omitempty"`
	AspectRatio string `json:"aspect_ratio,omitempty"`
	Resolution  string `json:"resolution,omitempty"`
}

// GenerateImageToVideo handles image-to-video generation via xAI Grok.
func (h *GrokHandler) GenerateImageToVideo(c *gin.Context) {
	var req GrokImageToVideoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	jobID, err := h.service.GenerateImageToVideo(c.Request.Context(), &service.GrokImageToVideoRequest{
		ImageURL:    req.ImageURL,
		Prompt:      req.Prompt,
		Duration:    req.Duration,
		AspectRatio: req.AspectRatio,
		Resolution:  req.Resolution,
	})
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "Failed to create job: "+err.Error())
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"success": true,
		"job_id":  jobID,
		"message": "Grok image-to-video job created",
	})
}

// GrokReferenceVideoRequest represents the HTTP request for reference-image video generation.
type GrokReferenceVideoRequest struct {
	Prompt             string   `json:"prompt" binding:"required"`
	ReferenceImageURLs []string `json:"reference_image_urls" binding:"required,min=1"`
	Duration           string   `json:"duration,omitempty"`
	AspectRatio        string   `json:"aspect_ratio,omitempty"`
	Resolution         string   `json:"resolution,omitempty"`
}

// GenerateReferenceVideo handles reference-image video generation via xAI Grok.
func (h *GrokHandler) GenerateReferenceVideo(c *gin.Context) {
	var req GrokReferenceVideoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	jobID, err := h.service.GenerateReferenceVideo(c.Request.Context(), &service.GrokReferenceVideoRequest{
		Prompt:             req.Prompt,
		ReferenceImageURLs: req.ReferenceImageURLs,
		Duration:           req.Duration,
		AspectRatio:        req.AspectRatio,
		Resolution:         req.Resolution,
	})
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "Failed to create job: "+err.Error())
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"success": true,
		"job_id":  jobID,
		"message": "Grok reference-video job created",
	})
}
