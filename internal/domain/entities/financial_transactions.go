package entities

import (
	"time"
)

type FinancialTransactions struct {
	Id             string    `gorm:"column:id;primaryKey"`
	WorkContractId string    `gorm:"column:work_contract_id"`
	Type           string    `gorm:"column:type"`
	Amount         int64     `gorm:"column:amount"`
	ReferenceId    *string   `gorm:"column:reference_id"`
	CreatedAt      time.Time `gorm:"column:created_at"`
}

func (FinancialTransactions) TableName() string { return "financial_transactions" }
