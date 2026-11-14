package entities

import (
	"time"
)

type WorkSubmissions struct {
	CollaborationId *string    `gorm:"column:collaboration_id"`
	VersionNo       *int32     `gorm:"column:version_no"`
	ApprovedAt      *time.Time `gorm:"column:approved_at"`
	Id              string     `gorm:"column:id;primaryKey"`
	WorkContractId  string     `gorm:"column:work_contract_id"`
	Description     string     `gorm:"column:description"`
	Status          string     `gorm:"column:status"`
	ReviewNote      *string    `gorm:"column:review_note"`
	SubmittedAt     time.Time  `gorm:"column:submitted_at"`
}

func (WorkSubmissions) TableName() string { return "work_submissions" }
