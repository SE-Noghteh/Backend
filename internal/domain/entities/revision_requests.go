package entities

import (
	"encoding/json"
	"time"
)

type RevisionRequests struct {
	Id            string           `gorm:"column:id;primaryKey"`
	DeliverableId string           `gorm:"column:deliverable_id"`
	Description   string           `gorm:"column:description"`
	Annotation    *json.RawMessage `gorm:"column:annotation"`
	Status        *string          `gorm:"column:status"`
	CreatedAt     *time.Time       `gorm:"column:created_at"`
}

func (RevisionRequests) TableName() string { return "revision_requests" }
