package entities

import (
	"time"
)

type Disputes struct {
	CollaborationId       *string    `gorm:"column:collaboration_id"`
	RespondBy             *time.Time `gorm:"column:respond_by"`
	Id                    string     `gorm:"column:id;primaryKey"`
	WorkContractId        string     `gorm:"column:work_contract_id"`
	CreatedByUserId       string     `gorm:"column:created_by_user_id"`
	AssignedSupportUserId *string    `gorm:"column:assigned_support_user_id"`
	Reason                string     `gorm:"column:reason"`
	Status                string     `gorm:"column:status"`
	Resolution            *string    `gorm:"column:resolution"`
	ResolutionNote        *string    `gorm:"column:resolution_note"`
	CreatedAt             time.Time  `gorm:"column:created_at"`
	ResolvedAt            *time.Time `gorm:"column:resolved_at"`
}

func (Disputes) TableName() string { return "disputes" }
