package entities

import (
	"time"
)

type FinancialAuditEvents struct {
	Id                     string    `gorm:"column:id;primaryKey"`
	FinancialTransactionId string    `gorm:"column:financial_transaction_id"`
	EventType              string    `gorm:"column:event_type"`
	DecisionRef            *string   `gorm:"column:decision_ref"`
	IdempotencyKey         string    `gorm:"column:idempotency_key;unique"`
	CreatedAt              time.Time `gorm:"column:created_at"`
}

func (FinancialAuditEvents) TableName() string { return "financial_audit_events" }
