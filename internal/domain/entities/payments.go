package entities

import (
	"time"
)

type Payments struct {
	Id               string     `gorm:"column:id;primaryKey"`
	WorkContractId   string     `gorm:"column:work_contract_id"`
	Amount           int64      `gorm:"column:amount"`
	Status           string     `gorm:"column:status"`
	GatewayReference *string    `gorm:"column:gateway_reference"`
	IdempotencyKey   string     `gorm:"column:idempotency_key"`
	PaidAt           *time.Time `gorm:"column:paid_at"`
	CreatedAt        time.Time  `gorm:"column:created_at"`
}

func (Payments) TableName() string { return "payments" }
