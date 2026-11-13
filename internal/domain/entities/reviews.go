package entities

import (
	"encoding/json"
)

type Reviews struct {
	Id              string           `gorm:"column:id;primaryKey"`
	CollaborationId string           `gorm:"column:collaboration_id"`
	AuthorUserId    string           `gorm:"column:author_user_id"`
	SubjectUserId   string           `gorm:"column:subject_user_id"`
	Rating          *int32           `gorm:"column:rating"`
	Criteria        *json.RawMessage `gorm:"column:criteria"`
	Comment         *string          `gorm:"column:comment"`
}

func (Reviews) TableName() string { return "reviews" }
