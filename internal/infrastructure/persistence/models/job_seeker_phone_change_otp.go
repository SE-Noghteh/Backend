package models

import "time"

// One challenge per account; a new request replaces the previous destination.
type JobSeekerPhoneChangeOTPModel struct {
	JobSeekerID uint64    `gorm:"primaryKey"`
	Phone       string    `gorm:"size:11;not null"`
	Salt        string    `gorm:"size:64;not null"`
	CodeHash    string    `gorm:"size:64;not null"`
	Attempts    int       `gorm:"not null"`
	SentAt      time.Time `gorm:"not null"`
	ExpiresAt   time.Time `gorm:"not null;index"`
}

func (JobSeekerPhoneChangeOTPModel) TableName() string { return "job_seeker_phone_change_otps" }
