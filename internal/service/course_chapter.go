package service

import (
	"context"
	"fmt"
	"time"

	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/database/models"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/jobs"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CourseChapterService handles chapter CRUD and plan execution
type CourseChapterService struct {
	db         *gorm.DB
	jobManager *jobs.Manager
}

// NewCourseChapterService creates a new CourseChapterService
func NewCourseChapterService(db *gorm.DB, jobManager *jobs.Manager) *CourseChapterService {
	return &CourseChapterService{db: db, jobManager: jobManager}
}

// CreateChapterRequest represents the input for creating a chapter
type CreateChapterRequest struct {
	Number  int            `json:"number"`
	Title   string         `json:"title"`
	Content map[string]any `json:"content,omitempty"`
}

// UpdateChapterRequest represents the input for updating a chapter
type UpdateChapterRequest struct {
	Number  *int           `json:"number,omitempty"`
	Title   *string        `json:"title,omitempty"`
	Content map[string]any `json:"content,omitempty"`
	Status  *string        `json:"status,omitempty"`
}

// Create creates a new chapter for a course
func (s *CourseChapterService) Create(ctx context.Context, courseID string, req *CreateChapterRequest) (*models.CourseChapter, error) {
	// Verify course exists
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.Course{}).Where("id = ?", courseID).Count(&count).Error; err != nil {
		return nil, fmt.Errorf("failed to verify course: %w", err)
	}
	if count == 0 {
		return nil, fmt.Errorf("course not found")
	}

	chapter := &models.CourseChapter{
		ID:        uuid.Must(uuid.NewV7()).String(),
		CourseID:  courseID,
		Number:    req.Number,
		Title:     req.Title,
		Content:   models.JSONB(req.Content),
		Status:    models.ChapterStatusDraft,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := s.db.WithContext(ctx).Create(chapter).Error; err != nil {
		return nil, fmt.Errorf("failed to create chapter: %w", err)
	}

	return chapter, nil
}

// GetByID retrieves a chapter by ID
func (s *CourseChapterService) GetByID(ctx context.Context, id string) (*models.CourseChapter, error) {
	var chapter models.CourseChapter
	if err := s.db.WithContext(ctx).First(&chapter, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("chapter not found: %w", err)
	}
	return &chapter, nil
}

// ListByCourse retrieves all chapters for a course
func (s *CourseChapterService) ListByCourse(ctx context.Context, courseID string) ([]*models.CourseChapter, error) {
	var chapters []*models.CourseChapter
	if err := s.db.WithContext(ctx).Where("course_id = ?", courseID).Order("number ASC").Find(&chapters).Error; err != nil {
		return nil, fmt.Errorf("failed to list chapters: %w", err)
	}
	return chapters, nil
}

// Update updates a chapter
func (s *CourseChapterService) Update(ctx context.Context, id string, req *UpdateChapterRequest) (*models.CourseChapter, error) {
	var chapter models.CourseChapter
	if err := s.db.WithContext(ctx).First(&chapter, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("chapter not found: %w", err)
	}

	if req.Number != nil {
		chapter.Number = *req.Number
	}
	if req.Title != nil {
		chapter.Title = *req.Title
	}
	if req.Content != nil {
		chapter.Content = models.JSONB(req.Content)
	}
	if req.Status != nil {
		chapter.Status = *req.Status
	}
	chapter.UpdatedAt = time.Now().UTC()

	if err := s.db.WithContext(ctx).Save(&chapter).Error; err != nil {
		return nil, fmt.Errorf("failed to update chapter: %w", err)
	}

	return &chapter, nil
}

// SavePlan saves a production plan for a chapter
func (s *CourseChapterService) SavePlan(ctx context.Context, id string, plan map[string]any) (*models.CourseChapter, error) {
	var chapter models.CourseChapter
	if err := s.db.WithContext(ctx).First(&chapter, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("chapter not found: %w", err)
	}

	chapter.Plan = models.JSONB(plan)
	chapter.Status = models.ChapterStatusPlanned
	chapter.UpdatedAt = time.Now().UTC()

	if err := s.db.WithContext(ctx).Save(&chapter).Error; err != nil {
		return nil, fmt.Errorf("failed to save plan: %w", err)
	}

	return &chapter, nil
}

// GetPlan retrieves the production plan for a chapter
func (s *CourseChapterService) GetPlan(ctx context.Context, id string) (map[string]any, error) {
	var chapter models.CourseChapter
	if err := s.db.WithContext(ctx).First(&chapter, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("chapter not found: %w", err)
	}
	return chapter.Plan, nil
}

// ExecutePlan executes AI jobs from a chapter's production plan
func (s *CourseChapterService) ExecutePlan(ctx context.Context, id string) ([]string, error) {
	var chapter models.CourseChapter
	if err := s.db.WithContext(ctx).First(&chapter, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("chapter not found: %w", err)
	}

	if chapter.Plan == nil {
		return nil, fmt.Errorf("chapter has no production plan")
	}

	// Extract segments from the plan
	segments, ok := chapter.Plan["segments"]
	if !ok {
		return nil, fmt.Errorf("plan has no segments")
	}

	segmentList, ok := segments.([]any)
	if !ok {
		return nil, fmt.Errorf("invalid segments format")
	}

	var jobIDs []string

	for _, seg := range segmentList {
		segment, ok := seg.(map[string]any)
		if !ok {
			continue
		}

		provider := mapSegmentToProvider(segment)
		if provider == "" {
			continue // Skip non-AI segments (e.g., recorded_video)
		}

		jobID, err := s.jobManager.CreateJobWithEntity(
			ctx,
			provider,
			segment,
			"course_chapter",
			chapter.ID,
		)
		if err != nil {
			return jobIDs, fmt.Errorf("failed to create job for segment: %w", err)
		}

		jobIDs = append(jobIDs, jobID)
	}

	// Update chapter status
	chapter.Status = models.ChapterStatusInProduction
	chapter.UpdatedAt = time.Now().UTC()
	_ = s.db.WithContext(ctx).Save(&chapter)

	return jobIDs, nil
}

// GetJobsForChapter returns all jobs linked to a chapter
func (s *CourseChapterService) GetJobsForChapter(ctx context.Context, chapterID string) ([]*models.Job, error) {
	return s.jobManager.ListJobsByEntity(ctx, "course_chapter", chapterID)
}

// mapSegmentToProvider maps a plan segment type to the appropriate AI provider
func mapSegmentToProvider(segment map[string]any) types.ProviderName {
	segType, _ := segment["type"].(string)
	provider, _ := segment["provider"].(string)

	// If provider is explicitly set, use it
	if provider != "" {
		switch provider {
		case "grok":
			return types.ProviderGrok
		case "grok-i2v":
			return types.ProviderGrokI2V
		case "grok-ref":
			return types.ProviderGrokRef
		case "elevenlabs":
			return types.ProviderElevenLabs
		case "nanobanana":
			return types.ProviderNanaBanana
		}
	}

	// Map by segment type
	switch segType {
	case "ai_image":
		return types.ProviderNanaBanana
	case "ai_video":
		return types.ProviderGrok
	case "speech", "voiceover":
		return types.ProviderElevenLabs
	default:
		return "" // recorded_video, etc. — no AI provider
	}
}
