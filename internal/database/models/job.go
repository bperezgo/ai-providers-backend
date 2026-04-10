package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/types"
)

// JSONB is a custom type for PostgreSQL JSONB columns
type JSONB map[string]any

// Value implements the driver.Valuer interface
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface
func (j *JSONB) Scan(value any) error {
	if value == nil {
		*j = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to unmarshal JSONB value")
	}

	return json.Unmarshal(bytes, j)
}

// Job represents an AI generation job
type Job struct {
	ID            string             `gorm:"primaryKey;type:uuid" json:"id"`
	Provider      types.ProviderName `gorm:"type:varchar(50);not null;index:idx_jobs_provider" json:"provider"`
	Status        string             `gorm:"type:varchar(20);not null;index:idx_jobs_status" json:"status"` // pending, processing, completed, failed
	Progress      int                `gorm:"type:integer;default:0" json:"progress"`
	ProviderJobID string             `gorm:"type:varchar(255)" json:"provider_job_id,omitempty"`
	RequestData   JSONB              `gorm:"type:jsonb" json:"request_data,omitempty"`
	ResultURL     string             `gorm:"type:text" json:"result_url,omitempty"`
	ResultData    JSONB              `gorm:"type:jsonb" json:"result_data,omitempty"`
	ErrorMessage  string             `gorm:"type:text" json:"error_message,omitempty"`
	EntityType    string             `gorm:"type:varchar(50);index:idx_jobs_entity" json:"entity_type,omitempty"`
	EntityID      *string            `gorm:"type:uuid;index:idx_jobs_entity" json:"entity_id,omitempty"`
	CreatedAt     time.Time          `gorm:"index:idx_jobs_created_at" json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
	CompletedAt   *time.Time         `json:"completed_at,omitempty"`
	ExpiresAt     time.Time          `gorm:"index:idx_jobs_expires_at" json:"expires_at"`
}

// TableName specifies the table name for the Job model
func (Job) TableName() string {
	return "jobs"
}

// Job status constants
const (
	JobStatusPending    = "pending"
	JobStatusProcessing = "processing"
	JobStatusCompleted  = "completed"
	JobStatusFailed     = "failed"
)

// IsTerminal returns true if the job is in a terminal state
func (j *Job) IsTerminal() bool {
	return j.Status == JobStatusCompleted || j.Status == JobStatusFailed
}

// IsPending returns true if the job is pending
func (j *Job) IsPending() bool {
	return j.Status == JobStatusPending
}

// IsProcessing returns true if the job is processing
func (j *Job) IsProcessing() bool {
	return j.Status == JobStatusProcessing
}

// IsCompleted returns true if the job completed successfully
func (j *Job) IsCompleted() bool {
	return j.Status == JobStatusCompleted
}

// IsFailed returns true if the job failed
func (j *Job) IsFailed() bool {
	return j.Status == JobStatusFailed
}
