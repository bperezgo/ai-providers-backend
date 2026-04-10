package mockserver

import (
	"encoding/json"
	"net/http"
)

// RegisterAnthropicRoutes registers Anthropic Claude mock endpoints.
func RegisterAnthropicRoutes(mux *http.ServeMux) {
	// Claude Messages API: POST /v1/messages
	mux.HandleFunc("POST /v1/messages", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		score := map[string]any{
			"prompt_adherence":  4,
			"artifact_score":    5,
			"composition_score": 4,
			"overall_score":     4,
			"reasoning":         "Mock evaluation: image meets quality standards for marketing use.",
			"suggested_retry":   false,
		}

		scoreJSON, _ := json.Marshal(score)

		json.NewEncoder(w).Encode(map[string]any{
			"id":    "msg_mock_001",
			"type":  "message",
			"role":  "assistant",
			"model": "claude-sonnet-4-6",
			"content": []map[string]any{
				{
					"type": "text",
					"text": string(scoreJSON),
				},
			},
			"stop_reason": "end_turn",
			"usage": map[string]any{
				"input_tokens":  100,
				"output_tokens": 50,
			},
		})
	})
}
