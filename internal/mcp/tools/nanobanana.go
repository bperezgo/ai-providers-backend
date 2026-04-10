package tools

import (
	"context"
	"encoding/json"

	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/service"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// RegisterNanoBananaTools registers NanoBanana MCP tools using the shared service.
func RegisterNanoBananaTools(s *mcpserver.MCPServer, svc *service.NanoBananaService) {
	s.AddTool(mcp.NewTool("nanobanana_generate_image",
		mcp.WithDescription("Generate high-quality images using Nano Banana Pro (via Fal.ai). Returns a job_id — poll with get_job_status, then retrieve the image path with get_job_result."),
		mcp.WithString("prompt", mcp.Required(), mcp.Description("Text description of the image to generate")),
		mcp.WithString("negative_prompt", mcp.Description("Elements to avoid in the generated image")),
		mcp.WithString("image_size", mcp.Description("Dimensions: 'square', 'square_hd', 'portrait_4_3', 'portrait_16_9', 'landscape_4_3', 'landscape_16_9' (default: square_hd)")),
		mcp.WithNumber("num_images", mcp.Description("Number of images 1-4 (default: 1)")),
		mcp.WithNumber("num_inference_steps", mcp.Description("Denoising steps (default: 28)")),
		mcp.WithNumber("guidance_scale", mcp.Description("How closely to follow the prompt 1-20 (default: 3.5)")),
		mcp.WithNumber("seed", mcp.Description("Random seed for reproducible results")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		prompt := mcp.ParseString(req, "prompt", "")
		if prompt == "" {
			return mcp.NewToolResultError("prompt is required"), nil
		}
		jobID, err := svc.GenerateImage(ctx, &service.NanoBananaRequest{
			Prompt:            prompt,
			NegativePrompt:    mcp.ParseString(req, "negative_prompt", ""),
			ImageSize:         mcp.ParseString(req, "image_size", ""),
			NumImages:         mcp.ParseInt(req, "num_images", 0),
			NumInferenceSteps: mcp.ParseInt(req, "num_inference_steps", 0),
			GuidanceScale:     mcp.ParseFloat64(req, "guidance_scale", 0),
			Seed:              int64(mcp.ParseInt64(req, "seed", 0)),
		})
		if err != nil {
			return mcp.NewToolResultError("Failed to create job: " + err.Error()), nil
		}
		out, _ := json.MarshalIndent(map[string]any{"job_id": jobID, "message": "Nano Banana image generation job created"}, "", "  ")
		return mcp.NewToolResultText(string(out)), nil
	})
}
