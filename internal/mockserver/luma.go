package mockserver

import (
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
)

// lumaQueue tracks Luma generation status.
type lumaQueue struct {
	mu       sync.Mutex
	polls    map[string]*atomic.Int32
	maxPolls int
}

func newLumaQueue(pollSteps int) *lumaQueue {
	return &lumaQueue{
		polls:    make(map[string]*atomic.Int32),
		maxPolls: pollSteps,
	}
}

func (q *lumaQueue) register(id string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.polls[id] = &atomic.Int32{}
}

func (q *lumaQueue) poll(id string) string {
	q.mu.Lock()
	counter, ok := q.polls[id]
	q.mu.Unlock()
	if !ok {
		return "completed"
	}
	n := counter.Add(1)
	if int(n) >= q.maxPolls {
		return "completed"
	}
	if n == 1 {
		return "queued"
	}
	return "processing"
}

// RegisterLumaRoutes registers Luma Labs mock endpoints.
func RegisterLumaRoutes(mux *http.ServeMux, cfg *Config, staticBaseURL string) {
	queue := newLumaQueue(cfg.PollSteps)

	// Submit generation: POST /dream-machine/v1/generations
	mux.HandleFunc("POST /dream-machine/v1/generations", func(w http.ResponseWriter, r *http.Request) {
		genID := uuid.Must(uuid.NewV7()).String()
		queue.register(genID)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"id":    genID,
			"state": "queued",
		})
	})

	// Poll status: GET /dream-machine/v1/generations/:id
	mux.HandleFunc("GET /dream-machine/v1/generations/", func(w http.ResponseWriter, r *http.Request) {
		genID := r.URL.Path[len("/dream-machine/v1/generations/"):]
		state := queue.poll(genID)

		resp := map[string]any{
			"id":    genID,
			"state": state,
		}

		if state == "completed" {
			resp["video"] = map[string]any{
				"url":      staticBaseURL + "/mock-files/sample.mp4",
				"width":    1920,
				"height":   1080,
				"duration": 5.0,
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
}
