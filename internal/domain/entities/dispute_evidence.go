package entities

import (
	"time"
)

type DisputeEvidence struct {
	Id               string    `gorm:"column:id;primaryKey"`
	DisputeId        string    `gorm:"column:dispute_id"`
	UploadedByUserId string    `gorm:"column:uploaded_by_user_id"`
	FileKey          string    `gorm:"column:file_key"`
	FileName         string    `gorm:"column:file_name"`
	MimeType         string    `gorm:"column:mime_type"`
	FileSize         int64     `gorm:"column:file_size"`
	Description      *string   `gorm:"column:description"`
	CreatedAt        time.Time `gorm:"column:created_at"`
}

func (DisputeEvidence) TableName() string { return "dispute_evidence" }
