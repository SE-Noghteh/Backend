package models

import "time"

type LoginAttemptModel struct {
	Phone           string    `gorm:"primaryKey;size:11"`
	Failures        int       `gorm:"not null"`
	WindowStartedAt time.Time `gorm:"not null;index"`
}

func (LoginAttemptModel) TableName() string { return "job_seeker_login_attempts" }
