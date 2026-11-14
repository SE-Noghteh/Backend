package entities

import (
	"time"
)

type Offers struct {
	Id               string     `gorm:"column:id;primaryKey"`
	JobApplicationId string     `gorm:"column:job_application_id"`
	ExpiresAt        time.Time  `gorm:"column:expires_at"`
	CreatedAt        *time.Time `gorm:"column:created_at"`
}

func (Offers) TableName() string { return "offers" }
