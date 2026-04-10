package port

import "context"

// EvalPersister can persist evaluation scores to a job's result data.
type EvalPersister interface {
	AppendJobResultData(ctx context.Context, jobID string, data map[string]any) error
}
