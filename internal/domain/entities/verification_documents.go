package entities

import (
	"time"
)

type VerificationDocuments struct {
	Id              string     `gorm:"column:id;primaryKey"`
	UserId          string     `gorm:"column:user_id"`
	DocumentType    string     `gorm:"column:document_type"`
	FileKey         string     `gorm:"column:file_key"`
	Status          string     `gorm:"column:status"`
	RejectionReason *string    `gorm:"column:rejection_reason"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	ReviewedAt      *time.Time `gorm:"column:reviewed_at"`
}

func (VerificationDocuments) TableName() string { return "verification_documents" }
