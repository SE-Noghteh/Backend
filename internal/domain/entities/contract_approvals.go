package entities

import (
	"time"
)

type ContractApprovals struct {
	Id              string     `gorm:"column:id;primaryKey"`
	WorkContractId  string     `gorm:"column:work_contract_id"`
	UserId          string     `gorm:"column:user_id"`
	Role            string     `gorm:"column:role"`
	Status          string     `gorm:"column:status"`
	ContractVersion int32      `gorm:"column:contract_version"`
	ApprovedAt      *time.Time `gorm:"column:approved_at"`
	RejectedAt      *time.Time `gorm:"column:rejected_at"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
}

func (ContractApprovals) TableName() string { return "contract_approvals" }
