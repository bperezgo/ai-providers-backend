package tools

import (
	"context"
	"encoding/json"

	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/providers/elevenlabs"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/service"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// RegisterSoundTools registers ElevenLabs MCP tools using the shared service.
func RegisterSoundTools(s *mcpserver.MCPServer, svc *service.SoundService) {
	registerTTS(s, svc)
	registerSoundEffects(s, svc)
	registerMusic(s, svc)
	registerGetVoices(s, svc)
}

func registerTTS(s *mcpserver.MCPServer, svc *service.SoundService) {
	s.AddTool(mcp.NewTool("elevenlabs_generate_speech",
		mcp.WithDescription("Generate text-to-speech audio using ElevenLabs. Returns a job_id — poll with get_job_status, then retrieve the audio file path with get_job_result."),
		mcp.WithString("text", mcp.Required(), mcp.Description("Text to convert to speech")),
		mcp.WithString("voice_id", mcp.Required(), mcp.Description("ElevenLabs voice ID. Use elevenlabs_get_voices to list available voices.")),
		mcp.WithString("model_id", mcp.Description("Model ID (default: eleven_monolingual_v1)")),
		mcp.WithNumber("stability", mcp.Description("Voice stability 0-1 (default: 0.5)")),
		mcp.WithNumber("similarity_boost", mcp.Description("Voice similarity boost 0-1 (default: 0.75)")),
		mcp.WithString("previous_text", mcp.Description("Text that came before the current text. Improves prosody continuity when generating sequential speech segments.")),
		mcp.WithString("next_text", mcp.Description("Text that comes after the current text. Improves prosody continuity when generating sequential speech segments.")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		text := mcp.ParseString(req, "text", "")
		voiceID := mcp.ParseString(req, "voice_id", "")
		if text == "" || voiceID == "" {
			return mcp.NewToolResultError("text and voice_id are required"), nil
		}
		ttsReq := &elevenlabs.TTSRequest{
			Text:         text,
			VoiceID:      voiceID,
			ModelID:      mcp.ParseString(req, "model_id", ""),
			PreviousText: mcp.ParseString(req, "previous_text", ""),
			NextText:     mcp.ParseString(req, "next_text", ""),
		}
		stability := mcp.ParseFloat64(req, "stability", -1)
		similarityBoost := mcp.ParseFloat64(req, "similarity_boost", -1)
		if stability >= 0 || similarityBoost >= 0 {
			ttsReq.Settings = &elevenlabs.VoiceSettings{}
			if stability >= 0 {
				ttsReq.Settings.Stability = stability
			}
			if similarityBoost >= 0 {
				ttsReq.Settings.SimilarityBoost = similarityBoost
			}
		}
		jobID, err := svc.GenerateSpeech(ctx, ttsReq)
		if err != nil {
			return mcp.NewToolResultError("Failed to create job: " + err.Error()), nil
		}
		out, _ := json.MarshalIndent(map[string]any{"job_id": jobID, "message": "Speech generation job created"}, "", "  ")
		return mcp.NewToolResultText(string(out)), nil
	})
}

func registerSoundEffects(s *mcpserver.MCPServer, svc *service.SoundService) {
	s.AddTool(mcp.NewTool("elevenlabs_generate_sound_effects",
		mcp.WithDescription("Generate sound effects from a text description using ElevenLabs. Returns a job_id."),
		mcp.WithString("text", mcp.Required(), mcp.Description("Text description of the sound effect")),
		mcp.WithNumber("duration_seconds", mcp.Description("Duration in seconds (0.5-22.0)")),
		mcp.WithNumber("prompt_influence", mcp.Description("Prompt influence 0-1 (default: 0.3)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		text := mcp.ParseString(req, "text", "")
		if text == "" {
			return mcp.NewToolResultError("text is required"), nil
		}
		jobID, err := svc.GenerateSoundEffects(ctx, &elevenlabs.SoundEffectRequest{
			Text:            text,
			DurationSeconds: mcp.ParseFloat64(req, "duration_seconds", 0),
			PromptInfluence: mcp.ParseFloat64(req, "prompt_influence", 0),
		})
		if err != nil {
			return mcp.NewToolResultError("Failed to create job: " + err.Error()), nil
		}
		out, _ := json.MarshalIndent(map[string]any{"job_id": jobID, "message": "Sound effect generation job created"}, "", "  ")
		return mcp.NewToolResultText(string(out)), nil
	})
}

func registerMusic(s *mcpserver.MCPServer, svc *service.SoundService) {
	s.AddTool(mcp.NewTool("elevenlabs_generate_music",
		mcp.WithDescription("Generate music from a text prompt using ElevenLabs. Returns a job_id."),
		mcp.WithString("text", mcp.Required(), mcp.Description("Text prompt describing the music")),
		mcp.WithNumber("duration_seconds", mcp.Description("Duration of the music in seconds")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		text := mcp.ParseString(req, "text", "")
		if text == "" {
			return mcp.NewToolResultError("text is required"), nil
		}
		jobID, err := svc.GenerateMusic(ctx, &elevenlabs.MusicRequest{
			Text:            text,
			DurationSeconds: mcp.ParseFloat64(req, "duration_seconds", 0),
		})
		if err != nil {
			return mcp.NewToolResultError("Failed to create job: " + err.Error()), nil
		}
		out, _ := json.MarshalIndent(map[string]any{"job_id": jobID, "message": "Music generation job created"}, "", "  ")
		return mcp.NewToolResultText(string(out)), nil
	})
}

func registerGetVoices(s *mcpserver.MCPServer, svc *service.SoundService) {
	s.AddTool(mcp.NewTool("elevenlabs_get_voices",
		mcp.WithDescription("List available ElevenLabs voices. Use the returned voice_id with elevenlabs_generate_speech."),
		mcp.WithString("language", mcp.Description("Language code to filter voices (e.g. 'en', 'es', 'pt')")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var opts *elevenlabs.GetVoicesOptions
		if lang := mcp.ParseString(req, "language", ""); lang != "" {
			opts = &elevenlabs.GetVoicesOptions{Language: lang}
		}
		voices, err := svc.GetVoices(ctx, opts)
		if err != nil {
			return mcp.NewToolResultError("Failed to retrieve voices: " + err.Error()), nil
		}
		out, _ := json.MarshalIndent(map[string]any{"voices": voices.Voices, "count": len(voices.Voices)}, "", "  ")
		return mcp.NewToolResultText(string(out)), nil
	})
}
