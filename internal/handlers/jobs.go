package handlers

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/jobs"
	"github.com/gin-gonic/gin"
)

// JobsHandler handles job-related requests
type JobsHandler struct {
	jobManager *jobs.Manager
}

// NewJobsHandler creates a new jobs handler
func NewJobsHandler(jobManager *jobs.Manager) *JobsHandler {
	return &JobsHandler{
		jobManager: jobManager,
	}
}

// GetStatus returns the current status of a job
func (h *JobsHandler) GetStatus(c *gin.Context) {
	jobID := c.Param("job_id")

	job, err := h.jobManager.GetJob(c.Request.Context(), jobID)
	if err != nil {
		errorResponse(c, http.StatusNotFound, "Job not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"job":     jobs.ToJobResponse(job),
	})
}

// GetResult returns the result of a completed job
func (h *JobsHandler) GetResult(c *gin.Context) {
	jobID := c.Param("job_id")

	job, err := h.jobManager.GetJob(c.Request.Context(), jobID)
	if err != nil {
		errorResponse(c, http.StatusNotFound, "Job not found")
		return
	}

	// Check if job is completed
	if !job.IsCompleted() {
		errorResponse(c, http.StatusBadRequest, fmt.Sprintf("Job is not completed (status: %s)", job.Status))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"job_id":     job.ID,
		"provider":   job.Provider,
		"result_url": job.ResultURL,
		"result":     job.ResultData,
	})
}

// StreamSSE streams job updates via Server-Sent Events
func (h *JobsHandler) StreamSSE(c *gin.Context) {
	jobID := c.Param("job_id")

	// Verify job exists
	job, err := h.jobManager.GetJob(c.Request.Context(), jobID)
	if err != nil {
		errorResponse(c, http.StatusNotFound, "Job not found")
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Subscribe to job events
	eventChan, cleanup := h.jobManager.Subscribe(jobID)
	defer cleanup()

	// Send initial status
	if err := h.sendSSEEvent(c.Writer, "status", jobs.ToJobResponse(job)); err != nil {
		return
	}
	c.Writer.Flush()

	// If job is already terminal, close connection
	if job.IsTerminal() {
		return
	}

	// Stream events
	timeout := time.After(10 * time.Minute)

	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				// Channel closed
				return
			}

			// Get updated job
			updatedJob, err := h.jobManager.GetJob(c.Request.Context(), jobID)
			if err != nil {
				h.sendSSEEvent(c.Writer, "error", gin.H{"error": "Failed to get job status"})
				return
			}

			// Send event based on type
			if err := h.sendSSEEvent(c.Writer, event.Type, jobs.ToJobResponse(updatedJob)); err != nil {
				return
			}
			c.Writer.Flush()

			// Close connection if job is terminal
			if updatedJob.IsTerminal() {
				return
			}

		case <-timeout:
			h.sendSSEEvent(c.Writer, "timeout", gin.H{"error": "Stream timeout"})
			return

		case <-c.Request.Context().Done():
			// Client disconnected
			return
		}
	}
}

// sendSSEEvent sends an SSE event
func (h *JobsHandler) sendSSEEvent(w io.Writer, eventType string, data interface{}) error {
	// Format: event: <type>\ndata: <json>\n\n
	event := fmt.Sprintf("event: %s\ndata: %v\n\n", eventType, data)
	if _, err := w.Write([]byte(event)); err != nil {
		return err
	}

	return nil
}

// ListJobs returns a list of jobs (optional filtering)
func (h *JobsHandler) ListJobs(c *gin.Context) {
	// Optional filters
	filters := make(map[string]interface{})

	if provider := c.Query("provider"); provider != "" {
		filters["provider"] = provider
	}
	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}
	if entityType := c.Query("entity_type"); entityType != "" {
		filters["entity_type"] = entityType
	}
	if entityID := c.Query("entity_id"); entityID != "" {
		filters["entity_id"] = entityID
	}

	limit := 100  // default limit
	offset := 0   // default offset

	jobsList, err := h.jobManager.ListJobs(c.Request.Context(), filters, limit, offset)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "Failed to list jobs: "+err.Error())
		return
	}

	var allJobs []interface{}
	for _, job := range jobsList {
		allJobs = append(allJobs, jobs.ToJobResponse(job))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"jobs":    allJobs,
		"count":   len(allJobs),
	})
}
