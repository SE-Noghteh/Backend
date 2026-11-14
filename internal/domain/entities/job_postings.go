package entities

import (
	"time"
)

type JobPostings struct {
	CategoryId       *string   `gorm:"column:category_id"`
	ActivityId       *string   `gorm:"column:activity_id"`
	CurrentVersionId *string   `gorm:"column:current_version_id"`
	Id               string    `gorm:"column:id;primaryKey"`
	EmployerId       string    `gorm:"column:employer_id"`
	Title            string    `gorm:"column:title"`
	Description      string    `gorm:"column:description"`
	Status           string    `gorm:"column:status"`
	CreatedAt        time.Time `gorm:"column:created_at"`
	UpdatedAt        time.Time `gorm:"column:updated_at"`
}

func (JobPostings) TableName() string { return "job_postings" }
