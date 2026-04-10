package handlers

import (
	"net/http"

	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/service"
	"github.com/gin-gonic/gin"
)

// CourseHandler handles course-related requests
type CourseHandler struct {
	service *service.CourseService
}

// NewCourseHandler creates a new course handler
func NewCourseHandler(svc *service.CourseService) *CourseHandler {
	return &CourseHandler{service: svc}
}

// CreateCourseHTTPRequest represents the HTTP request for creating a course
type CreateCourseHTTPRequest struct {
	Name        string         `json:"name" binding:"required"`
	Description string         `json:"description,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// Create handles course creation
func (h *CourseHandler) Create(c *gin.Context) {
	businessID := c.Param("business_id")

	var req CreateCourseHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	course, err := h.service.Create(c.Request.Context(), businessID, &service.CreateCourseRequest{
		Name:        req.Name,
		Description: req.Description,
		Metadata:    req.Metadata,
	})
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "Failed to create course: "+err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"course":  course,
	})
}

// Get handles retrieving a course by ID
func (h *CourseHandler) Get(c *gin.Context) {
	id := c.Param("course_id")

	course, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		errorResponse(c, http.StatusNotFound, "Course not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"course":  course,
	})
}

// ListByBusiness handles listing courses for a business
func (h *CourseHandler) ListByBusiness(c *gin.Context) {
	businessID := c.Param("business_id")

	courses, err := h.service.ListByBusiness(c.Request.Context(), businessID)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "Failed to list courses: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"courses": courses,
		"count":   len(courses),
	})
}

// Update handles updating a course
func (h *CourseHandler) Update(c *gin.Context) {
	id := c.Param("course_id")

	var req service.UpdateCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	course, err := h.service.Update(c.Request.Context(), id, &req)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "Failed to update course: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"course":  course,
	})
}

// Delete handles deleting a course
func (h *CourseHandler) Delete(c *gin.Context) {
	id := c.Param("course_id")

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		errorResponse(c, http.StatusNotFound, "Course not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Course deleted",
	})
}
