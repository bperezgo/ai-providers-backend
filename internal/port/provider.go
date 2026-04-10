package port

import "context"

// JobResult holds the standardized output from an AI provider.
type JobResult struct {
	ResultURL  string
	ResultData map[string]any
}

// AIClient defines the contract for AI provider clients used in job processing.
type AIClient[Req any] interface {
	Process(ctx context.Context, req *Req) (*JobResult, error)
}
