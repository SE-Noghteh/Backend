package entities

import (
	"time"
)

type CollaborationMessages struct {
	Id              string     `gorm:"column:id;primaryKey"`
	CollaborationId string     `gorm:"column:collaboration_id"`
	SenderUserId    string     `gorm:"column:sender_user_id"`
	Body            *string    `gorm:"column:body"`
	Hidden          bool       `gorm:"column:hidden"`
	ReadAt          *time.Time `gorm:"column:read_at"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
}

func (CollaborationMessages) TableName() string { return "collaboration_messages" }
