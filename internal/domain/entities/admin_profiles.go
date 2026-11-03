package entities

import (
	"time"
)

type AdminProfiles struct {
	UserId     string    `gorm:"column:user_id;primaryKey"`
	AdminScope string    `gorm:"column:admin_scope"`
	IsActive   bool      `gorm:"column:is_active"`
	CreatedAt  time.Time `gorm:"column:created_at"`
}

func (AdminProfiles) TableName() string { return "admin_profiles" }
