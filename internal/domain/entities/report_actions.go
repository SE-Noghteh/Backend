package entities

import (
	"time"
)

type ReportActions struct {
	Id          string     `gorm:"column:id;primaryKey"`
	ReportId    string     `gorm:"column:report_id"`
	AdminUserId string     `gorm:"column:admin_user_id"`
	Action      string     `gorm:"column:action"`
	Reason      *string    `gorm:"column:reason"`
	CreatedAt   *time.Time `gorm:"column:created_at"`
}

func (ReportActions) TableName() string { return "report_actions" }
