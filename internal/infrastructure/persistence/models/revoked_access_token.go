package models

import "time"

type RevokedAccessTokenModel struct {
	TokenID   string    `gorm:"primaryKey;size:32"`
	ExpiresAt time.Time `gorm:"not null;index"`
}

func (RevokedAccessTokenModel) TableName() string { return "revoked_access_tokens" }
