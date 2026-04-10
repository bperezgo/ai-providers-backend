package handlers

import (
	"net/http"
	"time"

	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/database"
	"github.com/gin-gonic/gin"
)

// HealthHandler handles health check requests
type HealthHandler struct{}

// NewHealthHandler creates a new health handler
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp string            `json:"timestamp"`
	Services  map[string]string `json:"services"`
}

// Check performs a health check
func (h *HealthHandler) Check(c *gin.Context) {
	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Services:  make(map[string]string),
	}

	// Check database health
	if err := database.HealthCheck(); err != nil {
		response.Status = "unhealthy"
		response.Services["database"] = "unhealthy: " + err.Error()
		_ = c.Error(err)
		c.JSON(http.StatusServiceUnavailable, response)
		return
	}

	response.Services["database"] = "healthy"

	c.JSON(http.StatusOK, response)
}
