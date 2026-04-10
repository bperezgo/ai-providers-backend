package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/jobs"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// RegisterJobTools registers get_job_status, get_job_result, and list_jobs tools.
func RegisterJobTools(s *mcpserver.MCPServer, jobManager *jobs.Manager) {
	s.AddTool(mcp.NewTool("get_job_status",
		mcp.WithDescription("Get the current status and progress of an async AI generation job."),
		mcp.WithString("job_id",
			mcp.Required(),
			mcp.Description("Job ID returned when the job was created"),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		jobID := mcp.ParseString(req, "job_id", "")
		if jobID == "" {
			return mcp.NewToolResultError("job_id is required"), nil
		}

		job, err := jobManager.GetJob(ctx, jobID)
		if err != nil {
			return mcp.NewToolResultError("Job not found: " + err.Error()), nil
		}

		out, _ := json.MarshalIndent(jobs.ToJobResponse(job), "", "  ")
		return mcp.NewToolResultText(string(out)), nil
	})

	s.AddTool(mcp.NewTool("get_job_result",
		mcp.WithDescription("Get the result of a completed AI generation job. Returns result URL, file path, and provider-specific data."),
		mcp.WithString("job_id",
			mcp.Required(),
			mcp.Description("Job ID of a completed job"),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		jobID := mcp.ParseString(req, "job_id", "")
		if jobID == "" {
			return mcp.NewToolResultError("job_id is required"), nil
		}

		job, err := jobManager.GetJob(ctx, jobID)
		if err != nil {
			return mcp.NewToolResultError("Job not found: " + err.Error()), nil
		}

		if !job.IsCompleted() {
			return mcp.NewToolResultError(fmt.Sprintf("Job is not completed yet (status: %s)", job.Status)), nil
		}

		out, _ := json.MarshalIndent(map[string]any{
			"job_id":     job.ID,
			"provider":   job.Provider,
			"result_url": job.ResultURL,
			"result":     job.ResultData,
		}, "", "  ")
		return mcp.NewToolResultText(string(out)), nil
	})

	s.AddTool(mcp.NewTool("list_jobs",
		mcp.WithDescription("List recent AI generation jobs with optional filtering by provider or status."),
		mcp.WithString("provider",
			mcp.Description("Filter by provider: elevenlabs, grok, nanobanana"),
		),
		mcp.WithString("status",
			mcp.Description("Filter by status: pending, processing, completed, failed"),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		filters := make(map[string]any)
		if p := mcp.ParseString(req, "provider", ""); p != "" {
			filters["provider"] = p
		}
		if st := mcp.ParseString(req, "status", ""); st != "" {
			filters["status"] = st
		}

		jobsList, err := jobManager.ListJobs(ctx, filters, 100, 0)
		if err != nil {
			return mcp.NewToolResultError("Failed to list jobs: " + err.Error()), nil
		}

		var all []any
		for _, j := range jobsList {
			all = append(all, jobs.ToJobResponse(j))
		}

		out, _ := json.MarshalIndent(map[string]any{"jobs": all, "count": len(all)}, "", "  ")
		return mcp.NewToolResultText(string(out)), nil
	})
}
