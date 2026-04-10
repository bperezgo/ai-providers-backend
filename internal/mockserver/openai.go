package mockserver

import (
	"encoding/json"
	"net/http"
	"time"
)

// RegisterOpenAIRoutes registers OpenAI/DALL-E mock endpoints.
func RegisterOpenAIRoutes(mux *http.ServeMux, staticBaseURL string) {
	// DALL-E image generation: POST /v1/images/generations
	mux.HandleFunc("POST /v1/images/generations", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"created": time.Now().Unix(),
			"data": []map[string]any{
				{
					"url":            staticBaseURL + "/mock-files/sample.png",
					"revised_prompt": "A mock image (revised by mock server)",
				},
			},
		})
	})
}
