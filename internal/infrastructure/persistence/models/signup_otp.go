package models

import "time"

type SignupOTPModel struct {
	Phone        string          `gorm:"primaryKey;size:11"`
	Salt         string          `gorm:"size:64;not null"`
	CodeHash     string          `gorm:"size:64;not null"`
	Attempts     int             `gorm:"not null"`
	SentAt       time.Time       `gorm:"not null"`
	ExpiresAt    time.Time       `gorm:"not null;index"`
	Registration *JobSeekerModel `gorm:"serializer:json;type:jsonb"`
}

func (SignupOTPModel) TableName() string { return "job_seeker_signup_otps" }
