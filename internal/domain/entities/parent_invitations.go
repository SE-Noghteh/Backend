package entities

import (
	"time"
)

type ParentInvitations struct {
	Id          string     `gorm:"column:id;primaryKey"`
	JobSeekerId string     `gorm:"column:job_seeker_id"`
	CodeHash    string     `gorm:"column:code_hash"`
	Status      string     `gorm:"column:status"`
	ExpiresAt   time.Time  `gorm:"column:expires_at"`
	UsedAt      *time.Time `gorm:"column:used_at"`
	ParentId    *string    `gorm:"column:parent_id"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
}

func (ParentInvitations) TableName() string { return "parent_invitations" }
