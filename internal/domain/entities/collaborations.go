package entities

import (
	"time"
)

type Collaborations struct {
	Id                      string     `gorm:"column:id;primaryKey"`
	WorkContractId          string     `gorm:"column:work_contract_id;unique"`
	EmployerId              string     `gorm:"column:employer_id"`
	JobSeekerId             string     `gorm:"column:job_seeker_id"`
	ActiveContractVersionId string     `gorm:"column:active_contract_version_id"`
	StartedAt               *time.Time `gorm:"column:started_at"`
	EndedAt                 *time.Time `gorm:"column:ended_at"`
	EndOutcome              *string    `gorm:"column:end_outcome"`
}

func (Collaborations) TableName() string { return "collaborations" }
