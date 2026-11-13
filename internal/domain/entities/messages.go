package entities

import (
	"time"
)

type Messages struct {
	Id             string     `gorm:"column:id;primaryKey"`
	ConversationId string     `gorm:"column:conversation_id"`
	SenderUserId   string     `gorm:"column:sender_user_id"`
	Content        *string    `gorm:"column:content"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
	ReadAt         *time.Time `gorm:"column:read_at"`
}

func (Messages) TableName() string { return "messages" }
