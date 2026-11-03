package entities

import (
	"time"
)

type ParentLinks struct {
	Id          string     `gorm:"column:id;primaryKey"`
	ParentId    string     `gorm:"column:parent_id"`
	JobSeekerId string     `gorm:"column:job_seeker_id"`
	InvitedAt   *time.Time `gorm:"column:invited_at"`
	AcceptedAt  *time.Time `gorm:"column:accepted_at"`
	RevokedAt   *time.Time `gorm:"column:revoked_at"`
}

func (ParentLinks) TableName() string { return "parent_links" }
