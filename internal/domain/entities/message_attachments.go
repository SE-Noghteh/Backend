package entities

import (
	"time"
)

type MessageAttachments struct {
	Id        string    `gorm:"column:id;primaryKey"`
	MessageId string    `gorm:"column:message_id"`
	FileKey   string    `gorm:"column:file_key"`
	FileName  string    `gorm:"column:file_name"`
	MimeType  string    `gorm:"column:mime_type"`
	FileSize  int64     `gorm:"column:file_size"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (MessageAttachments) TableName() string { return "message_attachments" }
