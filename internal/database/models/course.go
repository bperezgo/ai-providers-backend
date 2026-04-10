package models

import (
	"time"
)

// Course represents a course belonging to a business
type Course struct {
	ID          string    `gorm:"primaryKey;type:uuid" json:"id"`
	BusinessID  string    `gorm:"type:uuid;not null;index:idx_courses_business_id" json:"business_id"`
	Name        string    `gorm:"type:varchar(255);not null" json:"name"`
	Description string    `gorm:"type:text" json:"description,omitempty"`
	Status      string    `gorm:"type:varchar(50);not null;default:'draft'" json:"status"` // draft, active, archived
	Metadata    JSONB     `gorm:"type:jsonb" json:"metadata,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Associations
	Business Business `gorm:"foreignKey:BusinessID;constraint:OnDelete:CASCADE" json:"-"`
}

// TableName specifies the table name for the Course model
func (Course) TableName() string {
	return "courses"
}

// Course status constants
const (
	CourseStatusDraft    = "draft"
	CourseStatusActive   = "active"
	CourseStatusArchived = "archived"
)
