package models

import (
	"time"
)

// Business represents a business profile
type Business struct {
	ID          string    `gorm:"primaryKey;type:uuid" json:"id"`
	Name        string    `gorm:"type:varchar(255);not null" json:"name"`
	Slug        string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"slug"`
	Type        string    `gorm:"type:varchar(100)" json:"type,omitempty"`         // restaurant, course, personal-brand
	Location    string    `gorm:"type:varchar(500)" json:"location,omitempty"`
	Description string    `gorm:"type:text" json:"description,omitempty"`
	Discovery   JSONB     `gorm:"type:jsonb" json:"discovery,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName specifies the table name for the Business model
func (Business) TableName() string {
	return "businesses"
}
