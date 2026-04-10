package jobs

import (
	"context"
	"time"

	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/database/models"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/types"
)

// JobEvent represents an event broadcast via SSE
type JobEvent struct {
	JobID    string         `json:"job_id"`
	Type     string         `json:"type"` // status_update, completed, failed, progress
	Status   string         `json:"status,omitempty"`
	Progress int            `json:"progress,omitempty"`
	Data     map[string]any `json:"data,omitempty"`
}

// JobEventType constants
const (
	EventTypeStatusUpdate = "status_update"
	EventTypeCompleted    = "completed"
	EventTypeFailed       = "failed"
	EventTypeProgress     = "progress"
)

// CreateJobRequest represents a request to create a new job
type CreateJobRequest struct {
	Provider    string         `json:"provider"`
	RequestData map[string]any `json:"request_data"`
}

// UpdateJobRequest represents a request to update a job
type UpdateJobRequest struct {
	Status        string         `json:"status,omitempty"`
	Progress      int            `json:"progress,omitempty"`
	ProviderJobID string         `json:"provider_job_id,omitempty"`
	ResultURL     string         `json:"result_url,omitempty"`
	ResultData    map[string]any `json:"result_data,omitempty"`
	ErrorMessage  string         `json:"error_message,omitempty"`
}

// JobResponse represents a job in API responses
type JobResponse struct {
	ID            string             `json:"id"`
	Provider      types.ProviderName `json:"provider"`
	Status        string             `json:"status"`
	Progress      int                `json:"progress"`
	ProviderJobID string             `json:"provider_job_id,omitempty"`
	RequestData   map[string]any     `json:"request_data,omitempty"`
	ResultURL     string             `json:"result_url,omitempty"`
	ResultData    map[string]any     `json:"result_data,omitempty"`
	ErrorMessage  string             `json:"error_message,omitempty"`
	EntityType    string             `json:"entity_type,omitempty"`
	EntityID      string             `json:"entity_id,omitempty"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
	CompletedAt   *time.Time         `json:"completed_at,omitempty"`
	ExpiresAt     time.Time          `json:"expires_at"`
}

// ToJobResponse converts a Job model to a JobResponse
func ToJobResponse(job *models.Job) *JobResponse {
	return &JobResponse{
		ID:            job.ID,
		Provider:      job.Provider,
		Status:        job.Status,
		Progress:      job.Progress,
		ProviderJobID: job.ProviderJobID,
		RequestData:   job.RequestData,
		ResultURL:     job.ResultURL,
		ResultData:    job.ResultData,
		ErrorMessage:  job.ErrorMessage,
		EntityType: job.EntityType,
		EntityID: func() string {
			if job.EntityID != nil {
				return *job.EntityID
			}
			return ""
		}(),
		CreatedAt:     job.CreatedAt,
		UpdatedAt:     job.UpdatedAt,
		CompletedAt:   job.CompletedAt,
		ExpiresAt:     job.ExpiresAt,
	}
}

// Repository interface for job storage operations
type Repository interface {
	Create(ctx context.Context, job *models.Job) error
	GetByID(ctx context.Context, id string) (*models.Job, error)
	Update(ctx context.Context, job *models.Job) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filters map[string]any, limit int, offset int) ([]*models.Job, error)
	DeleteExpired(ctx context.Context, before time.Time) (int64, error)
}

// Broadcaster interface for SSE event broadcasting
type Broadcaster interface {
	Subscribe(jobID string) (<-chan JobEvent, func())
	Publish(event JobEvent)
	Close()
}
