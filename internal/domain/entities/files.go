package entities

import (
	"time"
)

type Files struct {
	Id         string     `gorm:"column:id;primaryKey"`
	OwnerId    string     `gorm:"column:owner_id"`
	Mime       string     `gorm:"column:mime"`
	SizeBytes  int64      `gorm:"column:size_bytes"`
	StorageKey string     `gorm:"column:storage_key"`
	CreatedAt  *time.Time `gorm:"column:created_at"`
}

func (Files) TableName() string { return "files" }
