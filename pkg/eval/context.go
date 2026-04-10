package eval

import "context"

type contextKey string

const jobIDKey contextKey = "eval_job_id"

// WithJobID returns a new context carrying the job ID for eval score persistence.
// Call this in the job processor before invoking the AI client.
func WithJobID(ctx context.Context, jobID string) context.Context {
	return context.WithValue(ctx, jobIDKey, jobID)
}

// JobIDFromContext extracts the job ID set by WithJobID.
// Returns ("", false) if no job ID is present.
func JobIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(jobIDKey).(string)
	return id, ok && id != ""
}
