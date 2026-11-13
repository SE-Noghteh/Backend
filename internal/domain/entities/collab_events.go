package entities

import (
	"encoding/json"
	"time"
)

type CollabEvents struct {
	Id              string           `gorm:"column:id;primaryKey"`
	CollaborationId string           `gorm:"column:collaboration_id"`
	ActorUserId     *string          `gorm:"column:actor_user_id"`
	EventType       string           `gorm:"column:event_type"`
	Payload         *json.RawMessage `gorm:"column:payload"`
	CreatedAt       *time.Time       `gorm:"column:created_at"`
}

func (CollabEvents) TableName() string { return "collab_events" }
