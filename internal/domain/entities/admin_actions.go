package entities

import (
	"encoding/json"
	"time"
)

type AdminActions struct {
	Id          string           `gorm:"column:id;primaryKey"`
	AdminUserId string           `gorm:"column:admin_user_id"`
	TargetType  string           `gorm:"column:target_type"`
	TargetId    string           `gorm:"column:target_id"`
	Action      string           `gorm:"column:action"`
	Reason      string           `gorm:"column:reason"`
	Changes     *json.RawMessage `gorm:"column:changes"`
	CreatedAt   *time.Time       `gorm:"column:created_at"`
}

func (AdminActions) TableName() string { return "admin_actions" }
