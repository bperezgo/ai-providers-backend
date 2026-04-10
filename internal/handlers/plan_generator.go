package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/service"
	"github.com/gin-gonic/gin"
)

// PlanGeneratorHandler handles plan generation endpoints.
type PlanGeneratorHandler struct {
	service *service.PlanGeneratorService
}

// NewPlanGeneratorHandler creates a new plan generator handler.
func NewPlanGeneratorHandler(svc *service.PlanGeneratorService) *PlanGeneratorHandler {
	return &PlanGeneratorHandler{service: svc}
}

// GeneratePlan handles POST /chapters/:chapter_id/generate-plan
// Streams progress via SSE while Claude generates the production plan.
func (h *PlanGeneratorHandler) GeneratePlan(c *gin.Context) {
	chapterID := c.Param("chapter_id")

	var req service.GeneratePlanRequest
	// Body is optional (feedback is optional)
	_ = c.ShouldBindJSON(&req)

	h.streamPlanGeneration(c, func(events chan<- service.PlanStreamEvent) error {
		return h.service.GeneratePlan(c.Request.Context(), chapterID, &req, events)
	})
}

// GenerateSection handles POST /chapters/:chapter_id/generate-section
// Regenerates a single section within an existing plan.
func (h *PlanGeneratorHandler) GenerateSection(c *gin.Context) {
	chapterID := c.Param("chapter_id")

	var req service.GenerateSectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	h.streamPlanGeneration(c, func(events chan<- service.PlanStreamEvent) error {
		return h.service.GenerateSection(c.Request.Context(), chapterID, &req, events)
	})
}

// RefineSegment handles POST /chapters/:chapter_id/refine-segment
// Refines a single segment with AI-powered feedback.
func (h *PlanGeneratorHandler) RefineSegment(c *gin.Context) {
	chapterID := c.Param("chapter_id")

	var req service.RefineSegmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	h.streamPlanGeneration(c, func(events chan<- service.PlanStreamEvent) error {
		return h.service.RefineSegment(c.Request.Context(), chapterID, &req, events)
	})
}

// streamPlanGeneration is a shared SSE streaming helper for all plan generation endpoints.
func (h *PlanGeneratorHandler) streamPlanGeneration(c *gin.Context, fn func(chan<- service.PlanStreamEvent) error) {
	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	events := make(chan service.PlanStreamEvent, 20)

	// Run generation in background
	errCh := make(chan error, 1)
	go func() {
		errCh <- fn(events)
	}()

	// Stream events to client
	flusher := c.Writer

	for event := range events {
		data, err := json.Marshal(event)
		if err != nil {
			continue
		}

		fmt.Fprintf(flusher, "event: %s\ndata: %s\n\n", event.Type, data)
		flusher.Flush()

		// Check if client disconnected
		select {
		case <-c.Request.Context().Done():
			return
		default:
		}
	}

	// Check for generation error
	if err := <-errCh; err != nil {
		errEvent, _ := json.Marshal(service.PlanStreamEvent{
			Type:    "error",
			Message: err.Error(),
		})
		fmt.Fprintf(flusher, "event: error\ndata: %s\n\n", errEvent)
		flusher.Flush()
	}
}
