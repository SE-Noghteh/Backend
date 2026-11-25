package models

import (
	"time"

	"zedlancer/backend/internal/domain/enums"
)

type PasswordResetModel struct {
	Role           enums.AccountRole `gorm:"size:24;primaryKey"`
	Phone          string            `gorm:"size:11;primaryKey"`
	AccountID      uint64            `gorm:"not null"`
	SessionVersion uint64            `gorm:"not null"`
	Salt           string            `gorm:"size:64;not null"`
	CodeHash       string            `gorm:"size:64;not null"`
	Attempts       int               `gorm:"not null"`
	SentAt         time.Time         `gorm:"not null"`
	ExpiresAt      time.Time         `gorm:"not null"`
	TokenHash      *string           `gorm:"size:64;uniqueIndex"`
}

func (PasswordResetModel) TableName() string { return "account_password_resets" }
