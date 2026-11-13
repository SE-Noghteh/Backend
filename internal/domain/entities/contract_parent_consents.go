package entities

import (
	"time"
)

type ContractParentConsents struct {
	Id                string     `gorm:"column:id;primaryKey"`
	ContractVersionId string     `gorm:"column:contract_version_id"`
	ParentId          string     `gorm:"column:parent_id"`
	JobSeekerId       string     `gorm:"column:job_seeker_id"`
	DenyReason        *string    `gorm:"column:deny_reason"`
	RespondBy         *time.Time `gorm:"column:respond_by"`
	DecidedAt         *time.Time `gorm:"column:decided_at"`
	WithdrawnAt       *time.Time `gorm:"column:withdrawn_at"`
}

func (ContractParentConsents) TableName() string { return "contract_parent_consents" }
