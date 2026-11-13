package entities

import (
	"time"
)

type UserSessions struct {
	Id         string     `gorm:"column:id;primaryKey"`
	UserId     string     `gorm:"column:user_id"`
	TokenHash  string     `gorm:"column:token_hash"`
	DeviceInfo *string    `gorm:"column:device_info"`
	CreatedAt  time.Time  `gorm:"column:created_at"`
	ExpiresAt  time.Time  `gorm:"column:expires_at"`
	RevokedAt  *time.Time `gorm:"column:revoked_at"`
}

func (UserSessions) TableName() string { return "user_sessions" }
