package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/service"
	"github.com/gin-gonic/gin"
)

// AIChatHandler handles the AI chat assistant endpoint.
type AIChatHandler struct {
	service *service.AIChatService
}

// NewAIChatHandler creates a new AI chat handler.
func NewAIChatHandler(svc *service.AIChatService) *AIChatHandler {
	return &AIChatHandler{service: svc}
}

// Chat handles POST /ai/chat — streams Claude's response via SSE.
func (h *AIChatHandler) Chat(c *gin.Context) {
	var req service.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if len(req.Messages) == 0 {
		errorResponse(c, http.StatusBadRequest, "At least one message is required")
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	events := make(chan service.ChatStreamEvent, 50)

	errCh := make(chan error, 1)
	go func() {
		errCh <- h.service.Chat(c.Request.Context(), &req, events)
	}()

	flusher := c.Writer

	for event := range events {
		data, err := json.Marshal(event)
		if err != nil {
			continue
		}
		fmt.Fprintf(flusher, "event: %s\ndata: %s\n\n", event.Type, data)
		flusher.Flush()

		select {
		case <-c.Request.Context().Done():
			return
		default:
		}
	}

	if err := <-errCh; err != nil {
		errEvent, _ := json.Marshal(service.ChatStreamEvent{
			Type:    "error",
			Content: err.Error(),
		})
		fmt.Fprintf(flusher, "event: error\ndata: %s\n\n", errEvent)
		flusher.Flush()
	}
}
