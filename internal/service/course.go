package service

import (
	"context"
	"fmt"
	"time"

	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/database/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CourseService handles course CRUD operations
type CourseService struct {
	db *gorm.DB
}

// NewCourseService creates a new CourseService
func NewCourseService(db *gorm.DB) *CourseService {
	return &CourseService{db: db}
}

// CreateCourseRequest represents the input for creating a course
type CreateCourseRequest struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// UpdateCourseRequest represents the input for updating a course
type UpdateCourseRequest struct {
	Name        *string        `json:"name,omitempty"`
	Description *string        `json:"description,omitempty"`
	Status      *string        `json:"status,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// Create creates a new course for a business
func (s *CourseService) Create(ctx context.Context, businessID string, req *CreateCourseRequest) (*models.Course, error) {
	// Verify business exists
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.Business{}).Where("id = ?", businessID).Count(&count).Error; err != nil {
		return nil, fmt.Errorf("failed to verify business: %w", err)
	}
	if count == 0 {
		return nil, fmt.Errorf("business not found")
	}

	course := &models.Course{
		ID:          uuid.Must(uuid.NewV7()).String(),
		BusinessID:  businessID,
		Name:        req.Name,
		Description: req.Description,
		Status:      models.CourseStatusDraft,
		Metadata:    models.JSONB(req.Metadata),
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := s.db.WithContext(ctx).Create(course).Error; err != nil {
		return nil, fmt.Errorf("failed to create course: %w", err)
	}

	return course, nil
}

// GetByID retrieves a course by ID
func (s *CourseService) GetByID(ctx context.Context, id string) (*models.Course, error) {
	var course models.Course
	if err := s.db.WithContext(ctx).First(&course, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("course not found: %w", err)
	}
	return &course, nil
}

// ListByBusiness retrieves all courses for a business
func (s *CourseService) ListByBusiness(ctx context.Context, businessID string) ([]*models.Course, error) {
	var courses []*models.Course
	if err := s.db.WithContext(ctx).Where("business_id = ?", businessID).Order("created_at DESC").Find(&courses).Error; err != nil {
		return nil, fmt.Errorf("failed to list courses: %w", err)
	}
	return courses, nil
}

// Update updates a course
func (s *CourseService) Update(ctx context.Context, id string, req *UpdateCourseRequest) (*models.Course, error) {
	var course models.Course
	if err := s.db.WithContext(ctx).First(&course, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("course not found: %w", err)
	}

	if req.Name != nil {
		course.Name = *req.Name
	}
	if req.Description != nil {
		course.Description = *req.Description
	}
	if req.Status != nil {
		course.Status = *req.Status
	}
	if req.Metadata != nil {
		course.Metadata = models.JSONB(req.Metadata)
	}
	course.UpdatedAt = time.Now().UTC()

	if err := s.db.WithContext(ctx).Save(&course).Error; err != nil {
		return nil, fmt.Errorf("failed to update course: %w", err)
	}

	return &course, nil
}

// Delete deletes a course
func (s *CourseService) Delete(ctx context.Context, id string) error {
	result := s.db.WithContext(ctx).Delete(&models.Course{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete course: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("course not found")
	}
	return nil
}
