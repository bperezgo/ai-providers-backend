package mockserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
)

// falQueue tracks in-flight requests for the Fal.ai queue simulation.
type falQueue struct {
	mu       sync.Mutex
	polls    map[string]*atomic.Int32 // requestID → poll count
	maxPolls int                      // polls before COMPLETED
}

func newFalQueue(pollSteps int) *falQueue {
	return &falQueue{
		polls:    make(map[string]*atomic.Int32),
		maxPolls: pollSteps,
	}
}

func (q *falQueue) register(id string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.polls[id] = &atomic.Int32{}
}

func (q *falQueue) poll(id string) string {
	q.mu.Lock()
	counter, ok := q.polls[id]
	q.mu.Unlock()
	if !ok {
		return "COMPLETED"
	}
	n := counter.Add(1)
	if int(n) >= q.maxPolls {
		return "COMPLETED"
	}
	if n == 1 {
		return "IN_QUEUE"
	}
	return "IN_PROGRESS"
}

// RegisterFalRoutes registers Fal.ai mock endpoints for NanoBanana, Pika, and Kling.
func RegisterFalRoutes(mux *http.ServeMux, cfg *Config, staticBaseURL string) {
	queue := newFalQueue(cfg.PollSteps)

	// Submit endpoints for all Fal.ai models
	submitHandler := func(w http.ResponseWriter, r *http.Request) {
		reqID := uuid.Must(uuid.NewV7()).String()
		queue.register(reqID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"request_id":   reqID,
			"status_url":   fmt.Sprintf("%s/fal-ai/queue/status/%s", staticBaseURL, reqID),
			"response_url": fmt.Sprintf("%s/fal-ai/queue/result/%s", staticBaseURL, reqID),
			"cancel_url":   fmt.Sprintf("%s/fal-ai/queue/cancel/%s", staticBaseURL, reqID),
		})
	}

	// NanoBanana
	mux.HandleFunc("POST /fal-ai/nano-banana-pro", submitHandler)
	// Pika
	mux.HandleFunc("POST /fal-ai/pika/v2.2/text-to-video", submitHandler)
	// Kling standard
	mux.HandleFunc("POST /fal-ai/kling-video/v1/standard/image-to-video", submitHandler)
	// Kling pro
	mux.HandleFunc("POST /fal-ai/kling-video/v1/pro/image-to-video", submitHandler)

	// Status polling — shared across all Fal.ai models
	mux.HandleFunc("GET /fal-ai/queue/status/", func(w http.ResponseWriter, r *http.Request) {
		reqID := r.URL.Path[len("/fal-ai/queue/status/"):]
		status := queue.poll(reqID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status": status,
		})
	})

	// Result — returns image result for NanoBanana, video result for Pika/Kling.
	// We return both fields; the client only reads what it needs.
	mux.HandleFunc("GET /fal-ai/queue/result/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"images": []map[string]any{
				{
					"url":          staticBaseURL + "/mock-files/sample.png",
					"content_type": "image/png",
					"width":        1024,
					"height":       1024,
					"file_size":    1024,
				},
			},
			"video": map[string]any{
				"url":          staticBaseURL + "/mock-files/sample.mp4",
				"content_type": "video/mp4",
				"file_name":    "mock-video.mp4",
				"file_size":    2048,
			},
		})
	})
}
