package models

import (
	"time"
	"zedlancer/backend/internal/domain/enums"
)

// Admin credentials have exactly two columns. Password contains a bcrypt hash.
type AdminModel struct {
	Username string `gorm:"primaryKey;size:100"`
	Password string `gorm:"size:60;not null"`
}

func (AdminModel) TableName() string { return "admins" }

type AdminLoginAttemptModel struct {
	Username        string    `gorm:"primaryKey;size:100"`
	Failures        int       `gorm:"not null"`
	WindowStartedAt time.Time `gorm:"not null"`
}

func (AdminLoginAttemptModel) TableName() string { return "admin_login_attempts" }

type AdminReviewModel struct {
	AdminApprovalStatus enums.AdminReviewStatus `gorm:"size:8;not null;default:pending;index"`
	AdminReviewMessage  string                  `gorm:"size:1000;not null;default:''"`
	AdminReviewedAt     *time.Time
	AdminReviewedBy     string `gorm:"size:100"`
}

type EmployerReviewMetadataModel struct {
	Message    string     `gorm:"column:admin_review_message;size:1000;not null;default:''"`
	ReviewedAt *time.Time `gorm:"column:admin_reviewed_at"`
	ReviewedBy string     `gorm:"column:admin_reviewed_by;size:100"`
}
