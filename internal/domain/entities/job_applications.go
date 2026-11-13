package entities

import (
	"time"
)

type JobApplications struct {
	Motivation     *string   `gorm:"column:motivation"`
	AvailableHours *int32    `gorm:"column:available_hours"`
	Id             string    `gorm:"column:id;primaryKey"`
	JobPostingId   string    `gorm:"column:job_posting_id"`
	JobSeekerId    string    `gorm:"column:job_seeker_id"`
	Status         string    `gorm:"column:status"`
	CoverLetter    *string   `gorm:"column:cover_letter"`
	CreatedAt      time.Time `gorm:"column:created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at"`
}

func (JobApplications) TableName() string { return "job_applications" }
