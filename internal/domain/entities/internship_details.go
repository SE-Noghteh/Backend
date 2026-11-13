package entities

import (
	"time"
)

type InternshipDetails struct {
	JobPostingVersionId string     `gorm:"column:job_posting_version_id;primaryKey"`
	LearningGoal        string     `gorm:"column:learning_goal"`
	PracticeSkills      *string    `gorm:"column:practice_skills"`
	Activities          *string    `gorm:"column:activities"`
	Prerequisites       *string    `gorm:"column:prerequisites"`
	StartDate           *time.Time `gorm:"column:start_date"`
	EndDate             *time.Time `gorm:"column:end_date"`
	ExpectedHours       *int32     `gorm:"column:expected_hours"`
	MentorName          string     `gorm:"column:mentor_name"`
	MentorContact       *string    `gorm:"column:mentor_contact"`
	Stipend             *float64   `gorm:"column:stipend"`
	CompletionCriteria  string     `gorm:"column:completion_criteria"`
}

func (InternshipDetails) TableName() string { return "internship_details" }
