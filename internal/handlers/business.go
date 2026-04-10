package handlers

import (
	"net/http"

	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/service"
	"github.com/gin-gonic/gin"
)

// BusinessHandler handles business-related requests
type BusinessHandler struct {
	service *service.BusinessService
}

// NewBusinessHandler creates a new business handler
func NewBusinessHandler(svc *service.BusinessService) *BusinessHandler {
	return &BusinessHandler{service: svc}
}

// CreateBusinessHTTPRequest represents the HTTP request for creating a business
type CreateBusinessHTTPRequest struct {
	Name        string         `json:"name" binding:"required"`
	Type        string         `json:"type,omitempty"`
	Location    string         `json:"location,omitempty"`
	Description string         `json:"description,omitempty"`
	Discovery   map[string]any `json:"discovery,omitempty"`
}

// Create handles business creation
func (h *BusinessHandler) Create(c *gin.Context) {
	var req CreateBusinessHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	business, err := h.service.Create(c.Request.Context(), &service.CreateBusinessRequest{
		Name:        req.Name,
		Type:        req.Type,
		Location:    req.Location,
		Description: req.Description,
		Discovery:   req.Discovery,
	})
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "Failed to create business: "+err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":  true,
		"business": business,
	})
}

// Get handles retrieving a business by ID
func (h *BusinessHandler) Get(c *gin.Context) {
	id := c.Param("business_id")

	business, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		errorResponse(c, http.StatusNotFound, "Business not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"business": business,
	})
}

// List handles listing all businesses
func (h *BusinessHandler) List(c *gin.Context) {
	businesses, err := h.service.List(c.Request.Context())
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "Failed to list businesses: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"businesses": businesses,
		"count":      len(businesses),
	})
}

// Update handles updating a business
func (h *BusinessHandler) Update(c *gin.Context) {
	id := c.Param("business_id")

	var req service.UpdateBusinessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	business, err := h.service.Update(c.Request.Context(), id, &req)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "Failed to update business: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"business": business,
	})
}

// Delete handles deleting a business
func (h *BusinessHandler) Delete(c *gin.Context) {
	id := c.Param("business_id")

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		errorResponse(c, http.StatusNotFound, "Business not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Business deleted",
	})
}
