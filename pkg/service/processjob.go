package service

import (
	"context"
	"fmt"
	"log"

	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/port"
	"github.com/bryanperez/laguna-escondida-marketing/backend/pkg/eval"
)

type JobProcessor[Req any] struct {
	jobsManager port.JobsManager
	client      port.AIClient[Req]
}

func NewJobProcessor[Req any](jobsManager port.JobsManager, client port.AIClient[Req]) JobProcessor[Req] {
	return JobProcessor[Req]{
		jobsManager: jobsManager,
		client:      client,
	}
}

func (jp *JobProcessor[Req]) Run(ctx context.Context, jobID string, req *Req) {
	// Detach from the caller's cancellation (e.g. HTTP request) while preserving
	// all values (trace IDs, etc.) so the async job runs to completion.
	ctx = eval.WithJobID(context.WithoutCancel(ctx), jobID)

	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic recovered in job %s: %v", jobID, r)
			if err := jp.jobsManager.UpdateJobWithError(ctx, jobID, fmt.Sprintf("internal panic: %v", r)); err != nil {
				log.Printf("Failed to update job with panic error: %v", err)
			}
		}
	}()

	if err := jp.jobsManager.UpdateJobStatus(ctx, jobID, "processing", 0); err != nil {
		log.Printf("Failed to update job status: %v", err)
		return
	}

	result, err := jp.client.Process(ctx, req)
	if err != nil {
		log.Printf("AI provider job %s failed: %v", jobID, err)
		if updateErr := jp.jobsManager.UpdateJobWithError(ctx, jobID, err.Error()); updateErr != nil {
			log.Printf("Failed to update job with error: %v", updateErr)
		}
		return
	}

	if err := jp.jobsManager.UpdateJobWithResult(ctx, jobID, result.ResultURL, result.ResultData); err != nil {
		log.Printf("Failed to update job with result: %v", err)
		return
	}

	log.Printf("Job %s completed successfully", jobID)
}
