package handlers

import (
	"net/http"

	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/jobs"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/service"
	"github.com/gin-gonic/gin"
)

// CourseChapterHandler handles chapter-related requests
type CourseChapterHandler struct {
	service *service.CourseChapterService
}

// NewCourseChapterHandler creates a new chapter handler
func NewCourseChapterHandler(svc *service.CourseChapterService) *CourseChapterHandler {
	return &CourseChapterHandler{service: svc}
}

// CreateChapterHTTPRequest represents the HTTP request for creating a chapter
type CreateChapterHTTPRequest struct {
	Number  int            `json:"number" binding:"required"`
	Title   string         `json:"title" binding:"required"`
	Content map[string]any `json:"content,omitempty"`
}

// Create handles chapter creation
func (h *CourseChapterHandler) Create(c *gin.Context) {
	courseID := c.Param("course_id")

	var req CreateChapterHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	chapter, err := h.service.Create(c.Request.Context(), courseID, &service.CreateChapterRequest{
		Number:  req.Number,
		Title:   req.Title,
		Content: req.Content,
	})
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "Failed to create chapter: "+err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"chapter": chapter,
	})
}

// Get handles retrieving a chapter by ID
func (h *CourseChapterHandler) Get(c *gin.Context) {
	id := c.Param("chapter_id")

	chapter, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		errorResponse(c, http.StatusNotFound, "Chapter not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"chapter": chapter,
	})
}

// ListByCourse handles listing chapters for a course
func (h *CourseChapterHandler) ListByCourse(c *gin.Context) {
	courseID := c.Param("course_id")

	chapters, err := h.service.ListByCourse(c.Request.Context(), courseID)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "Failed to list chapters: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"chapters": chapters,
		"count":    len(chapters),
	})
}

// Update handles updating a chapter
func (h *CourseChapterHandler) Update(c *gin.Context) {
	id := c.Param("chapter_id")

	var req service.UpdateChapterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	chapter, err := h.service.Update(c.Request.Context(), id, &req)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "Failed to update chapter: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"chapter": chapter,
	})
}

// SavePlan saves a production plan for a chapter
func (h *CourseChapterHandler) SavePlan(c *gin.Context) {
	id := c.Param("chapter_id")

	var plan map[string]any
	if err := c.ShouldBindJSON(&plan); err != nil {
		errorResponse(c, http.StatusBadRequest, "Invalid plan JSON: "+err.Error())
		return
	}

	chapter, err := h.service.SavePlan(c.Request.Context(), id, plan)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "Failed to save plan: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"chapter": chapter,
	})
}

// GetPlan retrieves the production plan for a chapter
func (h *CourseChapterHandler) GetPlan(c *gin.Context) {
	id := c.Param("chapter_id")

	plan, err := h.service.GetPlan(c.Request.Context(), id)
	if err != nil {
		errorResponse(c, http.StatusNotFound, "Chapter not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"plan":    plan,
	})
}

// ExecutePlan triggers AI job execution from a chapter's production plan
func (h *CourseChapterHandler) ExecutePlan(c *gin.Context) {
	id := c.Param("chapter_id")

	jobIDs, err := h.service.ExecutePlan(c.Request.Context(), id)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "Failed to execute plan: "+err.Error())
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"success": true,
		"job_ids": jobIDs,
		"count":   len(jobIDs),
		"message": "Plan execution started",
	})
}

// GetJobs returns all jobs linked to a chapter
func (h *CourseChapterHandler) GetJobs(c *gin.Context) {
	id := c.Param("chapter_id")

	jobModels, err := h.service.GetJobsForChapter(c.Request.Context(), id)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "Failed to get chapter jobs: "+err.Error())
		return
	}

	var jobResponses []any
	for _, j := range jobModels {
		jobResponses = append(jobResponses, jobs.ToJobResponse(j))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"jobs":    jobResponses,
		"count":   len(jobResponses),
	})
}
