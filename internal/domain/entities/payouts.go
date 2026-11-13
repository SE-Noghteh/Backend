package entities

import (
	"time"
)

type Payouts struct {
	Id             string     `gorm:"column:id;primaryKey"`
	WorkContractId string     `gorm:"column:work_contract_id"`
	JobSeekerId    string     `gorm:"column:job_seeker_id"`
	Amount         int64      `gorm:"column:amount"`
	Status         string     `gorm:"column:status"`
	IdempotencyKey string     `gorm:"column:idempotency_key"`
	PaidAt         *time.Time `gorm:"column:paid_at"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
}

func (Payouts) TableName() string { return "payouts" }
