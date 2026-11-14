package entities

import (
	"time"
)

type Deliverables struct {
	Id              string     `gorm:"column:id;primaryKey"`
	CollaborationId string     `gorm:"column:collaboration_id"`
	VersionNo       int32      `gorm:"column:version_no"`
	Note            *string    `gorm:"column:note"`
	SubmittedAt     *time.Time `gorm:"column:submitted_at"`
	ApprovedAt      *time.Time `gorm:"column:approved_at"`
}

func (Deliverables) TableName() string { return "deliverables" }
