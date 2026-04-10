package tools

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/service"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// RegisterGrokTools registers all xAI Grok video generation MCP tools.
func RegisterGrokTools(s *mcpserver.MCPServer, svc *service.GrokService) {
	registerGrokTextToVideo(s, svc)
	registerGrokImageToVideo(s, svc)
	registerGrokReferenceVideo(s, svc)
}

func registerGrokTextToVideo(s *mcpserver.MCPServer, svc *service.GrokService) {
	s.AddTool(mcp.NewTool("grok_generate_video",
		mcp.WithDescription("Generate a video from a text prompt using xAI Grok (grok-imagine-video). Optionally provide a source_image URL to guide the generation. Returns a job_id — poll with get_job_status, then retrieve the video with get_job_result."),
		mcp.WithString("prompt", mcp.Required(), mcp.Description("Text prompt describing the video to generate")),
		mcp.WithString("source_image", mcp.Description("Optional URL of a source image to guide the generation")),
		mcp.WithString("duration", mcp.Description("Video duration in seconds, 1–15 (default: '5')")),
		mcp.WithString("aspect_ratio", mcp.Description("Aspect ratio: '16:9', '9:16', '1:1', '4:3', '3:4', '3:2', '2:3' (default: '16:9')")),
		mcp.WithString("resolution", mcp.Description("Output resolution: '480p' or '720p' (default: '480p')")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		prompt := mcp.ParseString(req, "prompt", "")
		if prompt == "" {
			return mcp.NewToolResultError("prompt is required"), nil
		}
		jobID, err := svc.GenerateTextToVideo(ctx, &service.GrokTextToVideoRequest{
			Prompt:      prompt,
			SourceImage: mcp.ParseString(req, "source_image", ""),
			Duration:    mcp.ParseString(req, "duration", ""),
			AspectRatio: mcp.ParseString(req, "aspect_ratio", ""),
			Resolution:  mcp.ParseString(req, "resolution", ""),
		})
		if err != nil {
			return mcp.NewToolResultError("Failed to create job: " + err.Error()), nil
		}
		out, _ := json.MarshalIndent(map[string]any{"job_id": jobID, "message": "Grok text-to-video job created"}, "", "  ")
		return mcp.NewToolResultText(string(out)), nil
	})
}

func registerGrokImageToVideo(s *mcpserver.MCPServer, svc *service.GrokService) {
	s.AddTool(mcp.NewTool("grok_image_to_video",
		mcp.WithDescription("Animate a still image into a video using xAI Grok (grok-imagine-video). The source image becomes the first frame. Returns a job_id — poll with get_job_status, then retrieve the video with get_job_result."),
		mcp.WithString("image_url", mcp.Required(), mcp.Description("URL of the source image (becomes the first frame)")),
		mcp.WithString("prompt", mcp.Description("Optional text prompt to guide the animation")),
		mcp.WithString("duration", mcp.Description("Video duration in seconds, 1–15 (default: '5')")),
		mcp.WithString("aspect_ratio", mcp.Description("Aspect ratio: '16:9', '9:16', '1:1', '4:3', '3:4', '3:2', '2:3' (default: '16:9')")),
		mcp.WithString("resolution", mcp.Description("Output resolution: '480p' or '720p' (default: '480p')")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		imageURL := mcp.ParseString(req, "image_url", "")
		if imageURL == "" {
			return mcp.NewToolResultError("image_url is required"), nil
		}
		jobID, err := svc.GenerateImageToVideo(ctx, &service.GrokImageToVideoRequest{
			ImageURL:    imageURL,
			Prompt:      mcp.ParseString(req, "prompt", ""),
			Duration:    mcp.ParseString(req, "duration", ""),
			AspectRatio: mcp.ParseString(req, "aspect_ratio", ""),
			Resolution:  mcp.ParseString(req, "resolution", ""),
		})
		if err != nil {
			return mcp.NewToolResultError("Failed to create job: " + err.Error()), nil
		}
		out, _ := json.MarshalIndent(map[string]any{"job_id": jobID, "message": "Grok image-to-video job created"}, "", "  ")
		return mcp.NewToolResultText(string(out)), nil
	})
}

func registerGrokReferenceVideo(s *mcpserver.MCPServer, svc *service.GrokService) {
	s.AddTool(mcp.NewTool("grok_reference_video",
		mcp.WithDescription("Generate a video influenced by reference images using xAI Grok (grok-imagine-video). Reference images affect visual style without locking the first frame. Use <IMAGE_1>, <IMAGE_2>, etc. in the prompt to reference specific images. Returns a job_id — poll with get_job_status, then retrieve the video with get_job_result."),
		mcp.WithString("prompt", mcp.Required(), mcp.Description("Text prompt with <IMAGE_1>, <IMAGE_2> placeholders to reference images")),
		mcp.WithString("reference_image_urls", mcp.Required(), mcp.Description("Comma-separated list of reference image URLs")),
		mcp.WithString("duration", mcp.Description("Video duration in seconds, 1–15 (default: '5')")),
		mcp.WithString("aspect_ratio", mcp.Description("Aspect ratio: '16:9', '9:16', '1:1', '4:3', '3:4', '3:2', '2:3' (default: '16:9')")),
		mcp.WithString("resolution", mcp.Description("Output resolution: '480p' or '720p' (default: '480p')")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		prompt := mcp.ParseString(req, "prompt", "")
		if prompt == "" {
			return mcp.NewToolResultError("prompt is required"), nil
		}
		rawURLs := mcp.ParseString(req, "reference_image_urls", "")
		if rawURLs == "" {
			return mcp.NewToolResultError("reference_image_urls is required"), nil
		}
		var urls []string
		for _, u := range strings.Split(rawURLs, ",") {
			u = strings.TrimSpace(u)
			if u != "" {
				urls = append(urls, u)
			}
		}
		if len(urls) == 0 {
			return mcp.NewToolResultError("at least one reference image URL is required"), nil
		}
		jobID, err := svc.GenerateReferenceVideo(ctx, &service.GrokReferenceVideoRequest{
			Prompt:             prompt,
			ReferenceImageURLs: urls,
			Duration:           mcp.ParseString(req, "duration", ""),
			AspectRatio:        mcp.ParseString(req, "aspect_ratio", ""),
			Resolution:         mcp.ParseString(req, "resolution", ""),
		})
		if err != nil {
			return mcp.NewToolResultError("Failed to create job: " + err.Error()), nil
		}
		out, _ := json.MarshalIndent(map[string]any{"job_id": jobID, "message": "Grok reference-video job created"}, "", "  ")
		return mcp.NewToolResultText(string(out)), nil
	})
}
