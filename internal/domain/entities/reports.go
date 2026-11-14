package entities

import (
	"time"
)

type Reports struct {
	Id             string     `gorm:"column:id;primaryKey"`
	ReporterUserId string     `gorm:"column:reporter_user_id"`
	TargetType     string     `gorm:"column:target_type"`
	TargetId       string     `gorm:"column:target_id"`
	Reason         *string    `gorm:"column:reason"`
	Status         *string    `gorm:"column:status"`
	Priority       *string    `gorm:"column:priority"`
	CreatedAt      *time.Time `gorm:"column:created_at"`
}

func (Reports) TableName() string { return "reports" }
