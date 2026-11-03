package entities

import (
	"time"
)

type Conversations struct {
	Id               string     `gorm:"column:id;primaryKey"`
	JobApplicationId string     `gorm:"column:job_application_id;unique"`
	Status           string     `gorm:"column:status"`
	ClosedReason     *string    `gorm:"column:closed_reason"`
	ClosedAt         *time.Time `gorm:"column:closed_at"`
	CreatedAt        time.Time  `gorm:"column:created_at"`
	UpdatedAt        time.Time  `gorm:"column:updated_at"`
}

func (Conversations) TableName() string { return "conversations" }
