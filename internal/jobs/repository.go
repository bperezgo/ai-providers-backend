package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/database/models"
	"gorm.io/gorm"
)

// JobRepository implements the Repository interface using GORM
type JobRepository struct {
	db *gorm.DB
}

// NewJobRepository creates a new job repository
func NewJobRepository(db *gorm.DB) *JobRepository {
	return &JobRepository{db: db}
}

// Create inserts a new job into the database
func (r *JobRepository) Create(ctx context.Context, job *models.Job) error {
	if err := r.db.WithContext(ctx).Create(job).Error; err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}
	return nil
}

// GetByID retrieves a job by its ID
func (r *JobRepository) GetByID(ctx context.Context, id string) (*models.Job, error) {
	var job models.Job
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&job).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("job not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get job: %w", err)
	}
	return &job, nil
}

// Update updates an existing job
func (r *JobRepository) Update(ctx context.Context, job *models.Job) error {
	if err := r.db.WithContext(ctx).Save(job).Error; err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}
	return nil
}

// Delete removes a job from the database
func (r *JobRepository) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Delete(&models.Job{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}
	return nil
}

// List retrieves jobs with optional filters, limit, and offset
func (r *JobRepository) List(ctx context.Context, filters map[string]interface{}, limit int, offset int) ([]*models.Job, error) {
	var jobs []*models.Job

	query := r.db.WithContext(ctx).Model(&models.Job{})

	// Apply filters
	for key, value := range filters {
		query = query.Where(key+" = ?", value)
	}

	// Apply pagination
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	// Order by created_at descending
	query = query.Order("created_at DESC")

	if err := query.Find(&jobs).Error; err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	return jobs, nil
}

// DeleteExpired removes jobs that expired before the given time
func (r *JobRepository) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	result := r.db.WithContext(ctx).Where("expires_at < ?", before).Delete(&models.Job{})
	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete expired jobs: %w", result.Error)
	}
	return result.RowsAffected, nil
}

// GetByProvider retrieves jobs for a specific provider
func (r *JobRepository) GetByProvider(ctx context.Context, provider string, limit int) ([]*models.Job, error) {
	filters := map[string]interface{}{"provider": provider}
	return r.List(ctx, filters, limit, 0)
}

// GetByStatus retrieves jobs with a specific status
func (r *JobRepository) GetByStatus(ctx context.Context, status string, limit int) ([]*models.Job, error) {
	filters := map[string]interface{}{"status": status}
	return r.List(ctx, filters, limit, 0)
}

// GetPendingJobs retrieves all pending jobs
func (r *JobRepository) GetPendingJobs(ctx context.Context, limit int) ([]*models.Job, error) {
	return r.GetByStatus(ctx, models.JobStatusPending, limit)
}

// GetProcessingJobs retrieves all processing jobs
func (r *JobRepository) GetProcessingJobs(ctx context.Context, limit int) ([]*models.Job, error) {
	return r.GetByStatus(ctx, models.JobStatusProcessing, limit)
}

// CountByProvider counts jobs for a specific provider
func (r *JobRepository) CountByProvider(ctx context.Context, provider string) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.Job{}).Where("provider = ?", provider).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count jobs by provider: %w", err)
	}
	return count, nil
}

// CountByStatus counts jobs with a specific status
func (r *JobRepository) CountByStatus(ctx context.Context, status string) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.Job{}).Where("status = ?", status).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count jobs by status: %w", err)
	}
	return count, nil
}
