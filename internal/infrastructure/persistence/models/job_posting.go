package models

import (
	"encoding/json"
	"time"
)

type JobPostingModel struct {
	ID          uint64          `gorm:"primaryKey;autoIncrement"`
	EmployerID  uint64          `gorm:"not null;index"`
	Title       string          `gorm:"size:255;not null"`
	Description string          `gorm:"type:text;not null"`
	Details     json.RawMessage `gorm:"type:jsonb"`
	Status      string          `gorm:"size:30;not null;default:draft"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (JobPostingModel) TableName() string { return "job_postings" }
