package mockserver

import (
	"encoding/json"
	"net/http"
)

// RegisterElevenLabsRoutes registers ElevenLabs mock endpoints.
func RegisterElevenLabsRoutes(mux *http.ServeMux, staticBaseURL string) {
	// TTS: POST /v1/text-to-speech/:voiceId
	mux.HandleFunc("POST /v1/text-to-speech/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Write(sampleMP3())
	})

	// Sound effects: POST /v1/sound-generation
	mux.HandleFunc("POST /v1/sound-generation", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Write(sampleMP3())
	})

	// Music: POST /v1/music-generation
	mux.HandleFunc("POST /v1/music-generation", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Write(sampleMP3())
	})

	// Voices: GET /v1/voices
	mux.HandleFunc("GET /v1/voices", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"voices": []map[string]any{
				{
					"voice_id": "mock-voice-1",
					"name":     "Mock Voice (English)",
					"category": "premade",
					"labels":   map[string]string{"language": "en", "accent": "american"},
				},
				{
					"voice_id": "mock-voice-2",
					"name":     "Mock Voice (Spanish)",
					"category": "premade",
					"labels":   map[string]string{"language": "es", "accent": "latin"},
				},
				{
					"voice_id": "mock-voice-3",
					"name":     "Mock Voice (Portuguese)",
					"category": "premade",
					"labels":   map[string]string{"language": "pt", "accent": "brazilian"},
				},
			},
		})
	})
}
