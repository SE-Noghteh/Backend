package models

import "time"

type OTPChallengeModel struct {
	Phone     string    `gorm:"primaryKey;size:11"`
	Salt      string    `gorm:"size:64;not null"`
	CodeHash  string    `gorm:"size:64;not null"`
	Attempts  int       `gorm:"not null"`
	SentAt    time.Time `gorm:"not null"`
	ExpiresAt time.Time `gorm:"not null;index"`
}
