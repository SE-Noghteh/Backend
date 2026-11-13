package entities

import (
	"time"
)

type ContractMilestones struct {
	Id             string     `gorm:"column:id;primaryKey"`
	WorkContractId string     `gorm:"column:work_contract_id"`
	Title          string     `gorm:"column:title"`
	Description    *string    `gorm:"column:description"`
	Amount         int64      `gorm:"column:amount"`
	DueDate        *time.Time `gorm:"column:due_date"`
	Status         string     `gorm:"column:status"`
}

func (ContractMilestones) TableName() string { return "contract_milestones" }
