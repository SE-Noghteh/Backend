package entities

import (
	"time"
)

type Tasks struct {
	Id              string     `gorm:"column:id;primaryKey"`
	CollaborationId string     `gorm:"column:collaboration_id"`
	AssigneeUserId  *string    `gorm:"column:assignee_user_id"`
	Title           string     `gorm:"column:title"`
	Description     *string    `gorm:"column:description"`
	Status          *string    `gorm:"column:status"`
	DueAt           *time.Time `gorm:"column:due_at"`
}

func (Tasks) TableName() string { return "tasks" }
