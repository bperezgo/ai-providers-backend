package port

import "context"

// JobsManager defines the contract for managing a job's lifecycle during processing.
type JobsManager interface {
	UpdateJobStatus(ctx context.Context, jobID string, status string, progress int) error
	UpdateJobWithResult(ctx context.Context, jobID string, resultURL string, resultData map[string]any) error
	UpdateJobWithError(ctx context.Context, jobID string, errorMessage string) error
}
