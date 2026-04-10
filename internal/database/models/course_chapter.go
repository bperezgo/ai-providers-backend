package models

import (
	"time"
)

// CourseChapter represents a chapter within a course, including its production plan
type CourseChapter struct {
	ID        string    `gorm:"primaryKey;type:uuid" json:"id"`
	CourseID  string    `gorm:"type:uuid;not null;index:idx_course_chapters_course_id" json:"course_id"`
	Number    int       `gorm:"type:integer;not null" json:"number"`
	Title     string    `gorm:"type:varchar(255);not null" json:"title"`
	Content   JSONB     `gorm:"type:jsonb" json:"content,omitempty"` // Input: sections with narrative + visual direction
	Plan      JSONB     `gorm:"type:jsonb" json:"plan,omitempty"`    // Output: production plan with segments, AI calls
	Status    string    `gorm:"type:varchar(50);not null;default:'draft'" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Associations
	Course Course `gorm:"foreignKey:CourseID;constraint:OnDelete:CASCADE" json:"-"`
}

// TableName specifies the table name for the CourseChapter model
func (CourseChapter) TableName() string {
	return "course_chapters"
}

// Chapter status constants
const (
	ChapterStatusDraft        = "draft"
	ChapterStatusPlanned      = "planned"
	ChapterStatusInProduction = "in-production"
	ChapterStatusDone         = "done"
)
