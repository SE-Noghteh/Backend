package entities

import (
	"time"
)

type Notifications struct {
	Id              string     `gorm:"column:id;primaryKey"`
	RecipientUserId string     `gorm:"column:recipient_user_id"`
	Type            string     `gorm:"column:type"`
	LinkType        *string    `gorm:"column:link_type"`
	LinkId          *string    `gorm:"column:link_id"`
	ReadAt          *time.Time `gorm:"column:read_at"`
	CreatedAt       *time.Time `gorm:"column:created_at"`
}

func (Notifications) TableName() string { return "notifications" }
