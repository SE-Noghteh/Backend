package entities

import (
	"time"
)

type ParentLinkRequests struct {
	Id          string     `gorm:"column:id;primaryKey"`
	JobSeekerId string     `gorm:"column:job_seeker_id"`
	ParentId    string     `gorm:"column:parent_id"`
	Status      string     `gorm:"column:status"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
	RespondedAt *time.Time `gorm:"column:responded_at"`
}

func (ParentLinkRequests) TableName() string { return "parent_link_requests" }
