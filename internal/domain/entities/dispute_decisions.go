package entities

import (
	"time"
)

type DisputeDecisions struct {
	Id              string     `gorm:"column:id;primaryKey"`
	DisputeId       string     `gorm:"column:dispute_id"`
	AdminUserId     string     `gorm:"column:admin_user_id"`
	Decision        string     `gorm:"column:decision"`
	Reason          string     `gorm:"column:reason"`
	CollabEffect    *string    `gorm:"column:collab_effect"`
	FinancialEffect *string    `gorm:"column:financial_effect"`
	CreatedAt       *time.Time `gorm:"column:created_at"`
}

func (DisputeDecisions) TableName() string { return "dispute_decisions" }
