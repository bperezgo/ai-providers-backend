package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/database/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BusinessService handles business CRUD operations
type BusinessService struct {
	db *gorm.DB
}

// NewBusinessService creates a new BusinessService
func NewBusinessService(db *gorm.DB) *BusinessService {
	return &BusinessService{db: db}
}

// CreateBusinessRequest represents the input for creating a business
type CreateBusinessRequest struct {
	Name        string         `json:"name"`
	Type        string         `json:"type,omitempty"`
	Location    string         `json:"location,omitempty"`
	Description string         `json:"description,omitempty"`
	Discovery   map[string]any `json:"discovery,omitempty"`
}

// UpdateBusinessRequest represents the input for updating a business
type UpdateBusinessRequest struct {
	Name        *string        `json:"name,omitempty"`
	Type        *string        `json:"type,omitempty"`
	Location    *string        `json:"location,omitempty"`
	Description *string        `json:"description,omitempty"`
	Discovery   map[string]any `json:"discovery,omitempty"`
}

// Create creates a new business
func (s *BusinessService) Create(ctx context.Context, req *CreateBusinessRequest) (*models.Business, error) {
	business := &models.Business{
		ID:          uuid.Must(uuid.NewV7()).String(),
		Name:        req.Name,
		Slug:        generateSlug(req.Name),
		Type:        req.Type,
		Location:    req.Location,
		Description: req.Description,
		Discovery:   models.JSONB(req.Discovery),
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := s.db.WithContext(ctx).Create(business).Error; err != nil {
		return nil, fmt.Errorf("failed to create business: %w", err)
	}

	return business, nil
}

// GetByID retrieves a business by ID
func (s *BusinessService) GetByID(ctx context.Context, id string) (*models.Business, error) {
	var business models.Business
	if err := s.db.WithContext(ctx).First(&business, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("business not found: %w", err)
	}
	return &business, nil
}

// List retrieves all businesses
func (s *BusinessService) List(ctx context.Context) ([]*models.Business, error) {
	var businesses []*models.Business
	if err := s.db.WithContext(ctx).Order("created_at DESC").Find(&businesses).Error; err != nil {
		return nil, fmt.Errorf("failed to list businesses: %w", err)
	}
	return businesses, nil
}

// Update updates a business
func (s *BusinessService) Update(ctx context.Context, id string, req *UpdateBusinessRequest) (*models.Business, error) {
	var business models.Business
	if err := s.db.WithContext(ctx).First(&business, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("business not found: %w", err)
	}

	if req.Name != nil {
		business.Name = *req.Name
		business.Slug = generateSlug(*req.Name)
	}
	if req.Type != nil {
		business.Type = *req.Type
	}
	if req.Location != nil {
		business.Location = *req.Location
	}
	if req.Description != nil {
		business.Description = *req.Description
	}
	if req.Discovery != nil {
		business.Discovery = models.JSONB(req.Discovery)
	}
	business.UpdatedAt = time.Now().UTC()

	if err := s.db.WithContext(ctx).Save(&business).Error; err != nil {
		return nil, fmt.Errorf("failed to update business: %w", err)
	}

	return &business, nil
}

// Delete deletes a business
func (s *BusinessService) Delete(ctx context.Context, id string) error {
	result := s.db.WithContext(ctx).Delete(&models.Business{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete business: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("business not found")
	}
	return nil
}

var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)

// generateSlug creates a URL-friendly slug from a name
func generateSlug(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = nonAlphanumeric.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	return slug
}
