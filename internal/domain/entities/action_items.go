package entities

import (
	"time"
)

type ActionItems struct {
	Id          string     `gorm:"column:id;primaryKey"`
	OwnerUserId string     `gorm:"column:owner_user_id"`
	Kind        string     `gorm:"column:kind"`
	RefType     *string    `gorm:"column:ref_type"`
	RefId       *string    `gorm:"column:ref_id"`
	DueAt       *time.Time `gorm:"column:due_at"`
}

func (ActionItems) TableName() string { return "action_items" }
