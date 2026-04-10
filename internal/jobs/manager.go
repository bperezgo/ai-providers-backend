package jobs

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/database/models"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/types"
	"github.com/google/uuid"
)

// Manager handles job lifecycle and coordination
type Manager struct {
	repo        Repository
	broadcaster Broadcaster
	expiration  time.Duration
}

// NewManager creates a new job manager
func NewManager(repo Repository, broadcaster Broadcaster, expirationHours int) *Manager {
	return &Manager{
		repo:        repo,
		broadcaster: broadcaster,
		expiration:  time.Duration(expirationHours) * time.Hour,
	}
}

// CreateJob creates a new job and returns its ID
func (m *Manager) CreateJob(ctx context.Context, provider types.ProviderName, requestData map[string]any) (string, error) {
	job := &models.Job{
		ID:          uuid.Must(uuid.NewV7()).String(),
		Provider:    provider,
		Status:      models.JobStatusPending,
		Progress:    0,
		RequestData: models.JSONB(requestData),
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
		ExpiresAt:   time.Now().UTC().Add(m.expiration),
	}

	if err := m.repo.Create(ctx, job); err != nil {
		return "", fmt.Errorf("failed to create job: %w", err)
	}

	// Broadcast creation event
	m.publishEvent(JobEvent{
		JobID:  job.ID,
		Type:   EventTypeStatusUpdate,
		Status: job.Status,
		Data: map[string]any{
			"provider":   job.Provider,
			"created_at": job.CreatedAt,
		},
	})

	return job.ID, nil
}

// CreateJobWithEntity creates a new job linked to a specific entity (e.g., course_chapter)
func (m *Manager) CreateJobWithEntity(ctx context.Context, provider types.ProviderName, requestData map[string]any, entityType string, entityID string) (string, error) {
	job := &models.Job{
		ID:          uuid.Must(uuid.NewV7()).String(),
		Provider:    provider,
		Status:      models.JobStatusPending,
		Progress:    0,
		RequestData: models.JSONB(requestData),
		EntityType:  entityType,
		EntityID: func() *string {
			if entityID == "" {
				return nil
			}
			return &entityID
		}(),
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
		ExpiresAt:   time.Now().UTC().Add(m.expiration),
	}

	if err := m.repo.Create(ctx, job); err != nil {
		return "", fmt.Errorf("failed to create job: %w", err)
	}

	m.publishEvent(JobEvent{
		JobID:  job.ID,
		Type:   EventTypeStatusUpdate,
		Status: job.Status,
		Data: map[string]any{
			"provider":    job.Provider,
			"entity_type": entityType,
			"entity_id":   entityID,
			"created_at":  job.CreatedAt,
		},
	})

	return job.ID, nil
}

// ListJobsByEntity retrieves jobs linked to a specific entity
func (m *Manager) ListJobsByEntity(ctx context.Context, entityType string, entityID string) ([]*models.Job, error) {
	filters := map[string]any{
		"entity_type": entityType,
		"entity_id":   entityID,
	}
	return m.repo.List(ctx, filters, 100, 0)
}

// GetJob retrieves a job by ID
func (m *Manager) GetJob(ctx context.Context, jobID string) (*models.Job, error) {
	return m.repo.GetByID(ctx, jobID)
}

// UpdateJobStatus updates a job's status
func (m *Manager) UpdateJobStatus(ctx context.Context, jobID string, status string, progress int) error {
	job, err := m.repo.GetByID(ctx, jobID)
	if err != nil {
		return err
	}

	job.Status = status
	job.Progress = progress
	job.UpdatedAt = time.Now().UTC()

	// Set completion time if completed or failed
	if job.IsTerminal() && job.CompletedAt == nil {
		now := time.Now().UTC()
		job.CompletedAt = &now
	}

	if err := m.repo.Update(ctx, job); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// Broadcast status update
	eventType := EventTypeStatusUpdate
	if job.IsCompleted() {
		eventType = EventTypeCompleted
	} else if job.IsFailed() {
		eventType = EventTypeFailed
	}

	m.publishEvent(JobEvent{
		JobID:    jobID,
		Type:     eventType,
		Status:   status,
		Progress: progress,
	})

	return nil
}

// UpdateJobWithResult updates a job with completion data
func (m *Manager) UpdateJobWithResult(ctx context.Context, jobID string, resultURL string, resultData map[string]any) error {
	job, err := m.repo.GetByID(ctx, jobID)
	if err != nil {
		return err
	}

	job.Status = models.JobStatusCompleted
	job.Progress = 100
	job.ResultURL = resultURL
	job.ResultData = models.JSONB(resultData)
	job.UpdatedAt = time.Now().UTC()

	now := time.Now().UTC()
	job.CompletedAt = &now

	if err := m.repo.Update(ctx, job); err != nil {
		return fmt.Errorf("failed to update job with result: %w", err)
	}

	// Broadcast completion event
	m.publishEvent(JobEvent{
		JobID:    jobID,
		Type:     EventTypeCompleted,
		Status:   models.JobStatusCompleted,
		Progress: 100,
		Data: map[string]any{
			"result_url":  resultURL,
			"result_data": resultData,
		},
	})

	return nil
}

// UpdateJobWithError updates a job with error information
func (m *Manager) UpdateJobWithError(ctx context.Context, jobID string, errorMessage string) error {
	job, err := m.repo.GetByID(ctx, jobID)
	if err != nil {
		return err
	}

	job.Status = models.JobStatusFailed
	job.ErrorMessage = errorMessage
	job.UpdatedAt = time.Now().UTC()

	now := time.Now().UTC()
	job.CompletedAt = &now

	if err := m.repo.Update(ctx, job); err != nil {
		return fmt.Errorf("failed to update job with error: %w", err)
	}

	// Broadcast failure event
	m.publishEvent(JobEvent{
		JobID:  jobID,
		Type:   EventTypeFailed,
		Status: models.JobStatusFailed,
		Data: map[string]any{
			"error": errorMessage,
		},
	})

	return nil
}

// UpdateProgress updates only the progress of a job
func (m *Manager) UpdateProgress(ctx context.Context, jobID string, progress int) error {
	job, err := m.repo.GetByID(ctx, jobID)
	if err != nil {
		return err
	}

	job.Progress = progress
	job.UpdatedAt = time.Now().UTC()

	if err := m.repo.Update(ctx, job); err != nil {
		return fmt.Errorf("failed to update job progress: %w", err)
	}

	// Broadcast progress event
	m.publishEvent(JobEvent{
		JobID:    jobID,
		Type:     EventTypeProgress,
		Progress: progress,
	})

	return nil
}

// ListJobs retrieves jobs with filters
func (m *Manager) ListJobs(ctx context.Context, filters map[string]any, limit int, offset int) ([]*models.Job, error) {
	return m.repo.List(ctx, filters, limit, offset)
}

// DeleteExpiredJobs removes expired jobs and returns the count
func (m *Manager) DeleteExpiredJobs(ctx context.Context) (int64, error) {
	count, err := m.repo.DeleteExpired(ctx, time.Now().UTC())
	if err != nil {
		return 0, err
	}

	if count > 0 {
		log.Printf("Deleted %d expired jobs", count)
	}

	return count, nil
}

// Subscribe subscribes to job events via SSE
func (m *Manager) Subscribe(jobID string) (<-chan JobEvent, func()) {
	return m.broadcaster.Subscribe(jobID)
}

// publishEvent publishes an event to the broadcaster
func (m *Manager) publishEvent(event JobEvent) {
	m.broadcaster.Publish(event)
}

// AppendJobResultData merges data into the job's result_data JSONB column.
// Existing keys are preserved; keys in data are added or overwritten.
// Safe to call from a background goroutine after the job is already completed.
func (m *Manager) AppendJobResultData(ctx context.Context, jobID string, data map[string]any) error {
	job, err := m.repo.GetByID(ctx, jobID)
	if err != nil {
		return fmt.Errorf("AppendJobResultData: %w", err)
	}

	merged := make(map[string]any, len(job.ResultData)+len(data))
	for k, v := range job.ResultData {
		merged[k] = v
	}
	for k, v := range data {
		merged[k] = v
	}

	job.ResultData = models.JSONB(merged)
	job.UpdatedAt = time.Now().UTC()

	if err := m.repo.Update(ctx, job); err != nil {
		return fmt.Errorf("AppendJobResultData: %w", err)
	}
	return nil
}

// Close closes the manager and cleans up resources
func (m *Manager) Close() {
	m.broadcaster.Close()
}
