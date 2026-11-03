package entities

import (
	"time"
)

type StopRequests struct {
	Id                string     `gorm:"column:id;primaryKey"`
	CollaborationId   string     `gorm:"column:collaboration_id"`
	RequestedByUserId string     `gorm:"column:requested_by_user_id"`
	Reason            *string    `gorm:"column:reason"`
	Status            *string    `gorm:"column:status"`
	CreatedAt         *time.Time `gorm:"column:created_at"`
}

func (StopRequests) TableName() string { return "stop_requests" }
